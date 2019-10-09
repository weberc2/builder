package python

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

func virtualEnvParseInputs(
	inputs core.FrozenObject,
) ([]core.ArtifactID, error) {
	dependenciesValue, err := inputs.Get("dependencies")
	if err != nil {
		return nil, fmt.Errorf(
			"Missing required argument 'dependencies'",
		)
	}
	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return nil, fmt.Errorf(
			"'dependencies' argument must be a list",
		)
	}
	dependencies := make([]core.ArtifactID, len(dependenciesArray))
	for i, dependencyValue := range dependenciesArray {
		if dependency, ok := dependencyValue.(core.ArtifactID); ok {
			dependencies[i] = dependency
			continue
		}
		return nil, fmt.Errorf(
			"'dependencies' elements must be artifact IDs; found %T at "+
				"index %d",
			dependencyValue,
			i,
		)
	}
	return dependencies, nil
}

func installWheelPaths(
	path string,
	wheelPaths []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	concurrency := 4
	env := prependPATH(path)
	errs := make(chan error, concurrency)
	start := time.Now()

	// Launch `concurrency` workers
	for i := 0; i < concurrency; i++ {
		step := len(wheelPaths) / concurrency
		startIndex := i * step
		stopIndex := startIndex + step
		if i == concurrency-1 {
			stopIndex = len(wheelPaths)
		}
		go func(a, b int) {
			if b > len(wheelPaths) {
				b = len(wheelPaths)
			}

			// Run the `pip` in the virtualenv directory (passed in via
			// `path`). This should be sufficient to run this in the
			// virtualenv, but we're also updating the PATH environment
			// variable to include the `path` directory as well.
			cmd := exec.Command(
				filepath.Join(path, "pip"),
				append(
					[]string{"install", "--no-deps"}, wheelPaths[a:b]...,
				)...,
			)
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			cmd.Env = env
			errs <- cmd.Run()
		}(startIndex, stopIndex)
	}

	// Await each worker
	for i := 0; i < concurrency; i++ {
		if err := <-errs; err != nil {
			return err
		}
		log.Println(time.Since(start), i)
	}

	return nil
}

func deduplicate(sources []string) []string {
	seen := map[string]struct{}{}
	var output []string
	for _, source := range sources {
		if _, found := seen[source]; !found {
			output = append(output, source)
			seen[source] = struct{}{}
		}
	}
	return output
}

func findTarget(targets []core.DAG, id core.ArtifactID) (core.DAG, error) {
	for _, target := range targets {
		if target.ID.ArtifactID() == id {
			return target, nil
		}
	}
	return core.DAG{}, errors.Wrapf(ErrUnknownTarget, "Target = %s", id)
}

func gatherTargets(
	allTargets []core.DAG,
	ids []core.ArtifactID,
) ([]core.DAG, error) {
	var output []core.DAG
	for _, id := range ids {
		dag, err := findTarget(allTargets, id)
		if err != nil {
			return nil, err
		}
		output = append(output, dag)
	}
	return output, nil
}

func gatherWheelPaths(cache core.Cache, targets []core.DAG) ([]string, error) {
	var output []string
	for _, target := range targets {
		wheelPaths, err := fetchWheelPaths(cache, target)
		if err != nil {
			return nil, err
		}
		output = append(output, wheelPaths...)
	}
	return output, nil
}

func virtualEnvBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	dependencies, err := virtualEnvParseInputs(dag.Inputs)
	if err != nil {
		return err
	}
	return virtualEnvPrepare(dag, cache, stdout, stderr, dependencies)
}

func prependPATH(value string) []string {
	environ := os.Environ() // this is a copy, so no worries about mutating!
	for i, entry := range environ {
		if strings.HasPrefix(entry, "PATH=") {
			environ[i] = fmt.Sprintf(
				"PATH=%s:%s",
				value,
				entry[len("PATH="):],
			)
			return environ
		}
	}
	// If we never found the PATH env var in the list of env vars, then
	// create a new one and append it to the list
	return append(environ, "PATH="+value)
}

func replaceInFile(filePath, old, new string) error {
	fi, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(
		filePath,
		bytes.ReplaceAll(data, []byte(old), []byte(new)),
		fi.Mode(),
	)
}

// Moves the venv dir `old` to filepath `new`. Since venvs have files that
// contain their absolute paths, it's imperative to replace those references
// with references to the new absolute paths. As such, this script does that
// find and replace before moving the `old` dir to the `new` path, and this
// operation depends on `old` and `new` being absolute paths (this function
// does not verify, however). Also, an error can leave the `old` venv in a
// partially-moved state (references in the old files might be updated to their
// new paths); however, the new directory will never be created in a bad state.
func mvvenv(old, new string) error {
	// This function assumes that all files that _might_ contain references to
	// the old absolute path are included in this list.
	files := []string{
		"bin/easy_install",
		"bin/easy_install-3.6",
		"bin/pip",
		"bin/activate.fish",
		"bin/activate",
		"bin/activate.csh",
	}

	for _, file := range files {
		if err := replaceInFile(
			filepath.Join(old, file),
			old,
			new,
		); err != nil {
			return errors.Wrapf(
				err,
				"Replacing '%s' with '%s' in file '%s'",
				old,
				new,
				filepath.Join(old, file),
			)
		}
	}

	if err := os.Rename(old, new); err != nil {
		return errors.Wrapf(err, "Moving %s to %s", old, new)
	}

	return nil
}

func virtualEnvPrepare(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	dependencies []core.ArtifactID,
) error {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating temp dir")
	}
	defer os.Remove(tmpDir)

	venvDir := filepath.Join(tmpDir, ".venv")
	cmd := exec.Command("python", "-m", "venv", venvDir)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Creating virtualenv")
	}

	targets, err := gatherTargets(dag.Dependencies, dependencies)
	if err != nil {
		return err
	}
	wheelPaths, err := gatherWheelPaths(cache, targets)
	if err != nil {
		return err
	}
	wheelPaths = deduplicate(wheelPaths)

	if err := installWheelPaths(
		// Make sure `python` is the venv's python and not the system python.
		filepath.Join(venvDir, "bin"),
		wheelPaths,
		stdout,
		stderr,
	); err != nil {
		return errors.Wrapf(
			err,
			"Installing wheels [%s] into directory %s",
			strings.Join(wheelPaths, ", "),
			tmpDir,
		)
	}

	outputPath := cache.Path(dag.ID.ArtifactID())
	parentDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return errors.Wrap(
			err,
			"Making parent directory for output file in cache",
		)
	}

	// Since everything has succeeded, we can move the venv directory into
	// the cache.
	if err := mvvenv(venvDir, outputPath); err != nil {
		return errors.Wrapf(
			err,
			"Moving venv dir from %s to %s",
			venvDir,
			outputPath,
		)
	}

	return nil
}

var VirtualEnv = core.Plugin{
	Type:        BuilderTypeVirtualEnv,
	BuildScript: virtualEnvBuildScript,
}
