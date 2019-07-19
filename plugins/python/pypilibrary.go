package python

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

type pypiLibrary struct {
	packageName  string
	constraint   string
	dependencies []core.ArtifactID
}

func pypiLibraryInstall(
	output core.ArtifactID,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	lib pypiLibrary,
) error {
	cmd := exec.Command(
		"pip",
		"wheel",
		"--no-deps",
		"-w",
		cache.Path(output),
		lib.packageName+lib.constraint,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Installing pypi library")
	}
	return nil
}

func pypiLibraryParseInputs(inputs core.FrozenObject) (pypiLibrary, error) {
	packageNameValue, err := inputs.Get("package_name")
	if err != nil {
		return pypiLibrary{}, err
	}
	packageName, ok := packageNameValue.(core.String)
	if !ok {
		return pypiLibrary{}, errors.Errorf(
			"TypeError: package_name argument must be string; got %T",
			packageNameValue,
		)
	}

	constraintValue, err := inputs.Get("constraint")
	if err != nil {
		return pypiLibrary{}, err
	}
	constraint, ok := constraintValue.(core.String)
	if !ok {
		return pypiLibrary{}, errors.Errorf(
			"TypeError: constraint argument must be string; got %T",
			constraintValue,
		)
	}

	dependenciesValue, err := inputs.Get("dependencies")
	if err != nil {
		return pypiLibrary{}, fmt.Errorf(
			"Missing required argument 'dependencies'",
		)
	}
	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return pypiLibrary{}, fmt.Errorf(
			"'dependencies' argument must be a list",
		)
	}
	dependencies := make([]core.ArtifactID, len(dependenciesArray))
	for i, dependencyValue := range dependenciesArray {
		if dependency, ok := dependencyValue.(core.ArtifactID); ok {
			dependencies[i] = dependency
			continue
		}
		return pypiLibrary{}, fmt.Errorf(
			"'dependencies' elements must be artifact IDs; found %T at index %d",
			dependencyValue,
			i,
		)
	}
	return pypiLibrary{
		packageName:  string(packageName),
		constraint:   string(constraint),
		dependencies: dependencies,
	}, nil
}

func pypiLibraryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	lib, err := pypiLibraryParseInputs(dag.Inputs)
	if err != nil {
		return err
	}

	return pypiLibraryInstall(dag.ID.ArtifactID(), cache, stdout, stderr, lib)
}

var PypiLibrary = core.Plugin{
	Type: BuilderTypePypiLibrary,
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return pypiLibraryBuildScript, nil
	},
}
