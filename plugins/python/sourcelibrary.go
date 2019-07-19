package python

import (
	"fmt"
	"io"

	"github.com/weberc2/builder/core"
)

type sourceLibrary struct {
	packageName  string
	dependencies []core.ArtifactID
	sources      core.ArtifactID
}

func sourceLibraryInstall(
	output core.ArtifactID,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	lib sourceLibrary,
) error {
	return buildWheel(
		cache.Path(lib.sources),
		cache.Path(output),
		stdout,
		stderr,
	)
}

func sourceLibraryParseInputs(
	inputs core.FrozenObject,
) (sourceLibrary, error) {
	packageNameValue, err := inputs.Get("package_name")
	if err != nil {
		return sourceLibrary{}, err
	}
	packageName, ok := packageNameValue.(core.String)
	if !ok {
		return sourceLibrary{}, fmt.Errorf(
			"TypeError: package_name argument must be string; got %T",
			packageNameValue,
		)
	}

	dependenciesValue, err := inputs.Get("dependencies")
	if err != nil {
		return sourceLibrary{}, fmt.Errorf(
			"Missing required argument 'dependencies'",
		)
	}
	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return sourceLibrary{}, fmt.Errorf(
			"'dependencies' argument must be a list",
		)
	}
	dependencies := make([]core.ArtifactID, len(dependenciesArray))
	for i, dependencyValue := range dependenciesArray {
		if dependency, ok := dependencyValue.(core.ArtifactID); ok {
			dependencies[i] = dependency
			continue
		}
		return sourceLibrary{}, fmt.Errorf(
			"'dependencies' elements must be artifact IDs; found %T at index %d",
			dependencyValue,
			i,
		)
	}

	sourcesValue, err := inputs.Get("sources")
	if err != nil {
		return sourceLibrary{}, fmt.Errorf(
			"Missing required argument 'sources'",
		)
	}
	sources, ok := sourcesValue.(core.ArtifactID)
	if !ok {
		return sourceLibrary{}, fmt.Errorf(
			"'sources' argument must be a filegroup; got %T",
			sourcesValue,
		)
	}

	return sourceLibrary{
		packageName:  string(packageName),
		dependencies: dependencies,
		sources:      sources,
	}, nil
}

func sourceLibraryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	lib, err := sourceLibraryParseInputs(dag.Inputs)
	if err != nil {
		return err
	}

	return sourceLibraryInstall(
		dag.ID.ArtifactID(),
		cache,
		stdout,
		stderr,
		lib,
	)
}

var SourceLibrary = core.Plugin{
	Type: BuilderTypeSourceLibrary,
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return sourceLibraryBuildScript, nil
	},
}
