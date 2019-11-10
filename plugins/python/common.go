package python

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/weberc2/builder/core"
)

var ErrUnknownTarget = errors.New("Unknown target")

func fetchWheelPaths(cache core.Cache, dag core.DAG) ([]string, error) {
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

const (
	BuilderTypeSourceBinary  core.BuilderType = "py_source_binary"
	BuilderTypeSourceLibrary core.BuilderType = "py_source_library"
	BuilderTypePypiLibrary   core.BuilderType = "pypi"
	BuilderTypeTest          core.BuilderType = "pytest"
	BuilderTypeVirtualEnv    core.BuilderType = "virtualenv"
)

func isValidDependencyType(dependencyType core.BuilderType) bool {
	return dependencyType == BuilderTypeSourceLibrary ||
		dependencyType == BuilderTypePypiLibrary
}

var ErrInvalidDependencyType = errors.New(
	"Invalid builder type for Python target dependency",
)
