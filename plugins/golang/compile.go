package golang

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

func libraryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	link bool,
) error {
	packageName, err := dag.Inputs.GetString("package_name")
	if err != nil {
		return err
	}

	sourcesValue, err := dag.Inputs.Get("sources")
	if err != nil {
		return err
	}

	sources, ok := sourcesValue.(core.ArtifactID)
	if !ok {
		return errors.Errorf(
			"TypeError: wanted either filegroup or Go source target; got %T",
			sourcesValue,
		)
	}

	directory, err := dag.Inputs.GetString("directory")
	if err != nil {
		return err
	}

	dependenciesValue, err := dag.Inputs.Get("dependencies")
	if err != nil {
		return err
	}

	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return errors.Errorf(
			"TypeError: wanted list of Go compiled library targets; got %T",
			dependenciesValue,
		)
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating temporary directory")
	}
	defer os.Remove(tmpDir)

	targetFile := filepath.Join(tmpDir, string(packageName)+".a")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		return errors.Wrap(err, "Preparing temporary directory")
	}

	args := []string{"tool", "compile", "-pack", "-o", targetFile}

	// Add the dependency paths to the args list (-I flags)
	for i, v := range dependenciesArray {
		if dependency, ok := v.(core.ArtifactID); ok {
			args = append(args, "-I", cache.Path(dependency))
			continue
		}
		return errors.Errorf(
			"In 'dependencies' field, at index %d: TypeError: "+
				"wanted Go compiled library target; got %T",
			i,
			v,
		)
	}

	// Append all Go file paths in the sources filegroup to the arguments
	sourcesCacheDirectory := filepath.Join(
		cache.Path(sources),
		string(directory),
	)
	files, err := ioutil.ReadDir(sourcesCacheDirectory)
	if err != nil {
		return errors.Wrap(err, "Collecting go source files")
	}
	for _, file := range files {
		if !file.IsDir() &&
			(strings.HasSuffix(file.Name(), ".go") ||
				strings.HasSuffix(file.Name(), ".s")) &&
			//strings.HasSuffix(file.Name(), ".go") &&
			!strings.HasSuffix(file.Name(), "_test.go") {
			args = append(
				args,
				filepath.Join(sourcesCacheDirectory, file.Name()),
			)
		}
	}

	fmt.Fprintf(stderr, "Running: go %s\n", strings.Join(args, " "))
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=amd64")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Compiling source library")
	}

	finalCachePath := cache.Path(dag.ID.ArtifactID())
	if err := os.MkdirAll(filepath.Dir(finalCachePath), 0755); err != nil {
		return errors.Wrap(err, "Making parent directory in cache")
	}

	// if link is true, build the main executable and put the executable file
	// in the cache, otherwise move the archive into the cache.
	if link {
		dependencies, err := recursiveDependencies(dag)
		if err != nil {
			return errors.Wrap(err, "Recursively fetching dependencies")
		}

		args = []string{"tool", "link"}
		for _, dependency := range dependencies {
			args = append(args, "-L", cache.Path(dependency))
		}
		args = append(args, "-o", cache.Path(dag.ID.ArtifactID()), targetFile)
		fmt.Fprintf(stderr, "Running: go %s\n", strings.Join(args, " "))
		cmd := exec.Command("go", args...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "Compiling source library")
		}
	} else {
		if err := os.Rename(tmpDir, finalCachePath); err != nil {
			return errors.Wrap(err, "Moving temp dir into final cache location")
		}
	}

	return nil
}

func recursiveDependencies(dag core.DAG) ([]core.ArtifactID, error) {
	var output []core.ArtifactID

	dependenciesValue, err := dag.Inputs.Get("dependencies")
	if err != nil {
		return nil, errors.Wrapf(err, "Scanning dependencies on %s", dag.ID)
	}

	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return nil, errors.Errorf(
			"TypeError: expected list of Go compiled library targets; "+
				"found %T",
			dependenciesValue,
		)
	}

OUTER:
	for i, dependencyValue := range dependenciesArray {
		if dependencyID, ok := dependencyValue.(core.ArtifactID); ok {
			// Now that we know the current item in the dependency array is in
			// fact an ArtifactID, let's find the corresponding DAG in
			// dag.Dependencies such that we can recursively collect *its*
			// dependencies and attach them to `output`.
			for _, dependencyDAG := range dag.Dependencies {
				if dependencyDAG.ID.ArtifactID() == dependencyID {
					transitiveDeps, err := recursiveDependencies(dependencyDAG)
					if err != nil {
						return nil, errors.Wrapf(
							err,
							"Scanning dependency %s of %s",
							dependencyDAG.ID,
							dag.ID,
						)
					}

					// We've collected the dependencies of the current
					// dependencies; let's add them to `output` and move on to
					// the next dependency in `dependenciesArray`.
					output = append(output, transitiveDeps...)
					continue OUTER
				}
			}
			continue
		}
		return nil, errors.Errorf(
			"TypeError: Index %d: Expected Go compiled targets; found %T",
			i,
			dependencyValue,
		)
	}

	return append(output, dag.ID.ArtifactID()), nil
}

var Library = core.Plugin{
	Type: "go_library",
	Factory: func(core.FrozenObject) (core.BuildScript, error) {
		return func(
			dag core.DAG,
			cache core.Cache,
			stdout io.Writer,
			stderr io.Writer,
		) error {
			return libraryBuildScript(dag, cache, stdout, stderr, false)
		}, nil
	},
}

var Binary = core.Plugin{
	Type: "go_binary",
	Factory: func(core.FrozenObject) (core.BuildScript, error) {
		return func(
			dag core.DAG,
			cache core.Cache,
			stdout io.Writer,
			stderr io.Writer,
		) error {
			return libraryBuildScript(dag, cache, stdout, stderr, true)
		}, nil
	},
}
