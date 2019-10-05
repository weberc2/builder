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
	dir string,
	wheelPaths []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	cmd := exec.Command(
		"pip",
		append(
			[]string{
				"install",
				"--no-deps",
				"-t",
				dir,
			},
			wheelPaths...,
		)...,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", dir))
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Installing wheels to temp dir")
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
		tmpDir,
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

	// Since everything has succeeded, we can move the temporary directory into
	// the cache.
	if err := os.Rename(tmpDir, outputPath); err != nil {
		return errors.Wrap(err, "Moving temporary directory to cache")
	}

	return nil
}

var VirtualEnv = core.Plugin{
	Type:        BuilderTypeVirtualEnv,
	BuildScript: virtualEnvBuildScript,
}
