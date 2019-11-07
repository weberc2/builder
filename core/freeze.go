package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/pkg/errors"
)

func FreezeTarget(root string, cache Cache, target Target) (DAG, error) {
	return freezer.freezeTarget(
		freezer{root: root, cache: cache, seen: map[TargetID]DAG{}},
		target,
	)
}

type freezer struct {
	root  string
	cache Cache

	// An in-memory cache to make sure we don't redundantly freeze targets.
	seen map[TargetID]DAG
}

func (f freezer) freezeArray(a Array) ([]DAG, FrozenArray, error) {
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
	id, err := f.cache.TempDir(func(dir string) (string, ArtifactID, error) {
		checksums := []uint32{ChecksumString(string(fg.Package))}
		for _, pattern := range fg.Patterns {
			matches, err := doublestar.Glob(
				filepath.Join(f.root, string(fg.Package), pattern),
			)
			if err != nil {
				return "", ArtifactID{}, err
			}

			for _, match := range matches {
				data, err := ioutil.ReadFile(match)
				if err != nil {
					return "", ArtifactID{}, err
				}
				relpath, err := filepath.Rel(
					filepath.Join(f.root, string(fg.Package)),
					match,
				)
				if err != nil {
					return "", ArtifactID{}, err
				}
				checksums = append(
					checksums,
					JoinChecksums(
						ChecksumString(relpath),
						ChecksumBytes(data),
					),
				)

				if err := func() error {
					filePath := filepath.Join(dir, relpath)
					if err := os.MkdirAll(
						filepath.Dir(filePath),
						0755,
					); err != nil {
						return errors.Wrap(err, "Preparing parent directory")
					}

					return ioutil.WriteFile(filePath, data, 0644)
				}(); err != nil {
					return "", ArtifactID{}, errors.Wrapf(
						err,
						"Writing temp file for file %s in file group for "+
							"package %s",
						relpath,
						fg.Package,
					)
				}
			}
		}

		return "", ArtifactID{
			Package:  fg.Package,
			Checksum: JoinChecksums(checksums...),
		}, nil
	})

	if err != nil {
		return id, errors.Wrap(err, "Freezing file group")
	}

	return id, nil
}

var ErrTargetNotFound = errors.New("Target not found")

func (f freezer) freezeInput(i Input) ([]DAG, FrozenInput, error) {
	switch x := i.(type) {
	case Target:
		dag, err := f.freezeTarget(x)
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

func (f freezer) freezeTarget(t Target) (DAG, error) {
	if dag, found := f.seen[t.ID]; found {
		return dag, nil
	}

	deps, frozenInputs, err := f.freezeObject(t.Inputs)
	if err != nil {
		return DAG{}, err
	}

	dag := DAG{
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
		},
		Dependencies: deps,
	}
	f.seen[t.ID] = dag
	return dag, nil
}
