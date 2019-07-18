package python

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/weberc2/builder/core"
)

type sourceBinary struct {
	packageName  string
	entryPoint   string
	dependencies []core.ArtifactID
	sources      core.ArtifactID
}

var ErrUnknownTarget = errors.New("Unknown target")

func fetchWheelPaths(
	cache core.Cache,
	dag core.DAG,
) ([]string, error) {
	if !isValidDependencyType(dag.BuilderType) {
		return nil, errors.Wrapf(
			ErrInvalidDependencyType,
			"Target %s (type %s)",
			dag.ID,
			dag.BuilderType,
		)
	}

	dependenciesInput, err := dag.Inputs.Get("dependencies")
	if err != nil {
		panic(errors.Wrapf(
			err,
			"Trying to get input 'dependencies' on target %s (type %s)",
			dag.ID,
			dag.BuilderType,
		))
	}

	dependenciesArray, ok := dependenciesInput.(core.FrozenArray)
	if !ok {
		return nil, errors.Errorf(
			"Target %s: expected 'dependencies' input to be an array; got %T",
			dag.ID,
			dependenciesInput,
		)
	}

	var wheelPaths []string
DEPENDENCIES:
	for _, elt := range dependenciesArray {
		dependencyID, ok := elt.(core.ArtifactID)
		if !ok {
			return nil, errors.Errorf(
				"Target %s: expected dependency to be an artifact ID, got %T",
				dag.ID,
				elt,
			)
		}

		for _, dependency := range dag.Dependencies {
			if dependency.ID.ArtifactID() == dependencyID {
				transitiveWheelPaths, err := fetchWheelPaths(cache, dependency)
				if err != nil {
					return nil, errors.Wrapf(
						err,
						"Fetching dirs for dependency %s of target %s",
						dependencyID,
						dag.ID,
					)
				}

				wheelPaths = append(wheelPaths, transitiveWheelPaths...)
				continue DEPENDENCIES
			}
		}

		return nil, errors.Wrapf(
			ErrUnknownTarget,
			"Looking for dependency %s of target %s",
			dependencyID,
			dag.ID,
		)
	}

	wheelPath, err := fetchWheelPath(cache.Path(dag.ID.ArtifactID()))
	if err != nil {
		return nil, errors.Wrap(err, "Fetching wheel path")
	}

	return append(wheelPaths, wheelPath), nil
}

func fetchWheelPath(wheelDir string) (string, error) {
	// Since we can't pass an output filename to `pip wheel`, the best we can
	// do is give it an empty directory in which to write the wheel and then
	// hope when the command finishes we have a directory with a single file
	// (the wheel)
	files, err := ioutil.ReadDir(wheelDir)
	if err != nil {
		return "", errors.Wrap(err, "Reading the wheel directory")
	}
	if len(files) != 1 {
		fileNames := make([]string, len(files))
		for i, fileInfo := range files {
			fileNames[i] = fileInfo.Name()
			if fileInfo.IsDir() {
				fileNames[i] = fileNames[i] + "/"
			}
		}
		return "", errors.Errorf(
			"Expected the temp wheel directory contained 1 entry; found: [%s]",
			strings.Join(fileNames, ", "),
		)
	}
	return filepath.Join(wheelDir, files[0].Name()), nil
}

func sourceBinaryInstall(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	bin sourceBinary,
) error {
	tmpWheelDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating a temporary wheel dir")
	}
	defer os.Remove(tmpWheelDir)

	if err := buildWheel(
		cache.Path(bin.sources),
		tmpWheelDir,
		stdout,
		stderr,
	); err != nil {
		return errors.Wrap(err, "Creating wheel")
	}

	wheelPath, err := fetchWheelPath(tmpWheelDir)
	if err != nil {
		return errors.Wrap(err, "Fetching wheel path")
	}

	var wheelPaths []string
DEPENDENCIES:
	for _, dependency := range bin.dependencies {
		for _, target := range dag.Dependencies {
			if dependency == target.ID.ArtifactID() {
				targetWheelPaths, err := fetchWheelPaths(cache, target)
				if err != nil {
					return err
				}

				wheelPaths = append(wheelPaths, targetWheelPaths...)
				continue DEPENDENCIES
			}
		}
		return errors.Wrapf(ErrUnknownTarget, "Target = %s", dependency)
	}

	args := append(
		[]string{"--disable-cache", "--python", "python3.6", "--no-index"},
		append(wheelPaths, wheelPath)...,
	)

	args = append(
		args,
		"-o",
		cache.Path(dag.ID.ArtifactID()),
		"-e",
		fmt.Sprintf("%s:%s", bin.packageName, bin.entryPoint),
	)

	fmt.Fprintln(stdout, "Running command: pex", strings.Join(args, " "))
	cmd := exec.Command("pex", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(
			err,
			"Building pex for target %s",
			dag.ID.ArtifactID(),
		)
	}

	return nil
}

func buildWheel(
	sourcesDir string,
	outputDir string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.Wrap(err, "Creating the output dir")
	}

	cmd := exec.Command(
		"pip",
		"wheel",
		"--no-cache-dir",
		"-w",
		outputDir,
		sourcesDir,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(
			err,
			"Running 'pip wheel --no-cache-dir -w %s .' in workspace",
			outputDir,
		)
	}

	return nil
}

type sourceLibrary struct {
	packageName  string
	dependencies []core.ArtifactID
	sources      core.ArtifactID
}

func sourceLibraryInstall2(
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

type pypiLibrary struct {
	packageName  string
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
		lib.packageName,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Installing pypi library")
	}
	return nil
}

func sourceBinaryParseInputs(
	inputs core.FrozenObject,
) (sourceBinary, error) {
	packageNameValue, err := inputs.Get("package_name")
	if err != nil {
		return sourceBinary{}, err
	}
	packageName, ok := packageNameValue.(core.String)
	if !ok {
		return sourceBinary{}, fmt.Errorf(
			"TypeError: package_name argument must be string; got %T",
			packageNameValue,
		)
	}

	dependenciesValue, err := inputs.Get("dependencies")
	if err != nil {
		return sourceBinary{}, fmt.Errorf(
			"Missing required argument 'dependencies'",
		)
	}
	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return sourceBinary{}, fmt.Errorf(
			"'dependencies' argument must be a list",
		)
	}
	dependencies := make([]core.ArtifactID, len(dependenciesArray))
	for i, dependencyValue := range dependenciesArray {
		if dependency, ok := dependencyValue.(core.ArtifactID); ok {
			dependencies[i] = dependency
			continue
		}
		return sourceBinary{}, fmt.Errorf(
			"'dependencies' elements must be artifact IDs; found %T at index %d",
			dependencyValue,
			i,
		)
	}

	sourcesValue, err := inputs.Get("sources")
	if err != nil {
		return sourceBinary{}, fmt.Errorf(
			"Missing required argument 'sources'",
		)
	}
	sources, ok := sourcesValue.(core.ArtifactID)
	if !ok {
		return sourceBinary{}, fmt.Errorf(
			"'sources' argument must be a filegroup; got %T",
			sourcesValue,
		)
	}

	entryPointValue, err := inputs.Get("entry_point")
	if err != nil {
		return sourceBinary{}, err
	}
	entryPoint, ok := entryPointValue.(core.String)
	if !ok {
		return sourceBinary{}, fmt.Errorf(
			"TypeError: entry_point argument must be string; got %T",
			entryPointValue,
		)
	}

	return sourceBinary{
		packageName:  string(packageName),
		dependencies: dependencies,
		sources:      sources,
		entryPoint:   string(entryPoint),
	}, nil
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
		dependencies: dependencies,
	}, nil
}

func sourceBinaryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	bin, err := sourceBinaryParseInputs(dag.Inputs)
	if err != nil {
		return err
	}
	return sourceBinaryInstall(dag, cache, stdout, stderr, bin)
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

	return sourceLibraryInstall2(
		dag.ID.ArtifactID(),
		cache,
		stdout,
		stderr,
		lib,
	)
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

const (
	BuilderTypeSourceBinary  core.BuilderType = "py_source_binary"
	BuilderTypeSourceLibrary core.BuilderType = "py_source_library"
	BuilderTypePypiLibrary   core.BuilderType = "py_pypi_library"
)

func isValidDependencyType(dependencyType core.BuilderType) bool {
	return dependencyType == BuilderTypeSourceLibrary ||
		dependencyType == BuilderTypePypiLibrary
}

var ErrInvalidDependencyType = errors.New(
	"Invalid builder type for Python target dependency",
)

var SourceBinary = core.Plugin{
	Type: BuilderTypeSourceBinary,
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return sourceBinaryBuildScript, nil
	},
}

var SourceLibrary = core.Plugin{
	Type: BuilderTypeSourceLibrary,
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return sourceLibraryBuildScript, nil
	},
}

var PypiLibrary = core.Plugin{
	Type: BuilderTypePypiLibrary,
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return pypiLibraryBuildScript, nil
	},
}
