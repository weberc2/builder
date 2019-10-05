package python

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
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

var SourceBinary = core.Plugin{
	Type:        BuilderTypeSourceBinary,
	BuildScript: sourceBinaryBuildScript,
}
