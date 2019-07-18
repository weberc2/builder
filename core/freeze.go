package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func FreezeTargetID(
	root string,
	cache Cache,
	evaluator Evaluator,
	targetID TargetID,
) (DAG, error) {
	f := freezer{
		root:      root,
		cache:     cache,
		evaluator: evaluator,
	}
	return f.freezeTargetID(targetID)
}

type freezer struct {
	root      string
	evaluator Evaluator
	cache     Cache
}

func (f *freezer) freezeArray(a Array) ([]DAG, FrozenArray, error) {
	var deps []DAG
	out := make(FrozenArray, len(a))
	for i, elt := range a {
		dependencies, frozenElt, err := f.freezeInput(elt)
		if err != nil {
			return nil, nil, err
		}
		out[i] = frozenElt
		deps = append(deps, dependencies...)
	}
	return deps, out, nil
}

func (f *freezer) freezeFileGroup(fg FileGroup) (ArtifactID, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return ArtifactID{}, errors.Wrap(err, "Creating temporary directory")
	}
	committed := false

	defer func() {
		if !committed {
			if err := os.Remove(tmpDir); err != nil {
				log.Printf("ERROR removing temporary directory %s", tmpDir)
			}
		}
	}()

	checksums := make([]uint32, len(fg.Paths)+1)
	checksums[0] = ChecksumString(string(fg.Package))
	for i, path := range fg.Paths {
		data, err := ioutil.ReadFile(filepath.Join(
			f.root,
			string(fg.Package),
			path,
		))
		if err != nil {
			return ArtifactID{}, err
		}

		checksums[i+1] = JoinChecksums(
			ChecksumString(path),
			ChecksumBytes(data),
		)

		if err := func() error {
			filePath := filepath.Join(tmpDir, path)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return errors.Wrap(err, "Preparing parent directory")
			}
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer func() {
				if err := file.Close(); err != nil {
					log.Printf("ERROR closing file %s", filePath)
				}
			}()

			if _, err := file.Write(data); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			return ArtifactID{}, errors.Wrapf(
				err,
				"Writing temp file for file %s in file group for package %s",
				path,
				fg.Package,
			)
		}
	}

	aid := ArtifactID{
		Package:  fg.Package,
		Checksum: JoinChecksums(checksums...),
	}

	if err := os.MkdirAll(filepath.Dir(f.cache.Path(aid)), 0755); err != nil {
		return ArtifactID{}, errors.Wrapf(err, "Preparing directory in cache")
	}
	if err := os.RemoveAll(f.cache.Path(aid)); err != nil {
		return ArtifactID{}, errors.Wrapf(err, "Removing old cache directory")
	}
	if err := os.Rename(tmpDir, f.cache.Path(aid)); err != nil {
		return ArtifactID{}, errors.Wrap(err, "Committing temp dir to cache")
	}
	committed = true

	return aid, nil
}

var ErrTargetNotFound = errors.New("Target not found")

func (f *freezer) freezeTargetID(tid TargetID) (DAG, error) {
	targets, err := f.evaluator.Evaluate(tid.Package)
	if err != nil {
		return DAG{}, err
	}

	for _, target := range targets {
		if target.ID == tid {
			dag, err := f.freezeTarget(target)
			if err != nil {
				return DAG{}, errors.Wrapf(err, "Freezing target %s", tid)
			}
			return dag, nil
		}
	}

	return DAG{}, errors.Wrapf(ErrTargetNotFound, "Target %s", tid)
}

func (f *freezer) freezeInput(i Input) ([]DAG, FrozenInput, error) {
	switch x := i.(type) {
	case TargetID:
		dag, err := f.freezeTargetID(x)
		if err != nil {
			return nil, nil, err
		}
		return []DAG{dag}, dag.ID.ArtifactID(), nil
	case FileGroup:
		artifactID, err := f.freezeFileGroup(x)
		if err != nil {
			return nil, ArtifactID{}, errors.Wrapf(
				err,
				"Freezing source path %s",
				x,
			)
		}
		return nil, artifactID, nil
	case Int:
		return nil, x, nil
	case String:
		return nil, x, nil
	case Bool:
		return nil, x, nil
	case Object:
		return f.freezeObject(x)
	case Array:
		return f.freezeArray(x)
	case nil:
		return nil, nil, nil
	}
	panic(fmt.Sprintf("Invalid input type: %T", i))
}

func (f *freezer) freezeObject(o Object) ([]DAG, FrozenObject, error) {
	var deps []DAG
	out := make(FrozenObject, len(o))
	for i, field := range o {
		dependencies, frozenValue, err := f.freezeInput(field.Value)
		if err != nil {
			return nil, nil, err
		}

		out[i] = FrozenField{Key: field.Key, Value: frozenValue}
		deps = append(deps, dependencies...)
	}
	return deps, out, nil
}

func (f *freezer) freezeTarget(t Target) (DAG, error) {
	deps, frozenInputs, err := f.freezeObject(t.Inputs)
	if err != nil {
		return DAG{}, err
	}

	return DAG{
		FrozenTarget: FrozenTarget{
			ID: FrozenTargetID{
				Package: t.ID.Package,
				Target:  t.ID.Target,
				Checksum: JoinChecksums(
					ChecksumString(string(t.ID.Package)),
					ChecksumString(string(t.ID.Target)),
					ChecksumString(string(t.BuilderType)),
					frozenInputs.checksum(),
					// TODO: Checksum the builder args
				),
			},
			Inputs:      frozenInputs,
			BuilderType: t.BuilderType,
			BuilderArgs: nil, // TODO: Freeze builder args?
		},
		Dependencies: deps,
	}, nil
}
