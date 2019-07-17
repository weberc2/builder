package core

import (
	"fmt"
	"io"
	"io/ioutil"
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

func (f *freezer) freezeSourcePath(sp SourcePath) (ArtifactID, error) {
	data, err := ioutil.ReadFile(filepath.Join(
		f.root,
		string(sp.Package),
		sp.FilePath,
	))
	if err != nil {
		return ArtifactID{}, err
	}

	aid := ArtifactID{
		FrozenTargetID: FrozenTargetID{
			Package: sp.Package,
			Target:  sp.Target,
			Checksum: JoinChecksums(
				ChecksumString(string(sp.Package)),
				ChecksumString(string(sp.Target)),
				ChecksumString(sp.FilePath),
				ChecksumBytes(data),
			),
		},
		FilePath: sp.FilePath,
	}

	return aid, f.cache.Write(aid, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
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
	case SourcePath:
		artifactID, err := f.freezeSourcePath(x)
		if err != nil {
			return nil, ArtifactID{}, errors.Wrapf(
				err,
				"Reading source path %s",
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
