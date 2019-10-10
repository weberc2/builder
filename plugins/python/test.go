package python

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

type test struct {
	directory    string
	dependencies core.ArtifactID
	sources      core.ArtifactID
}

func testParseInputs(inputs core.FrozenObject) (test, error) {
	directoryValue, err := inputs.Get("directory")
	if err != nil {
		return test{}, fmt.Errorf(
			"Missing required argument 'directory'",
		)
	}
	directory, ok := directoryValue.(core.String)
	if !ok {
		return test{}, fmt.Errorf(
			"TypeError: 'directory' argument must be a str",
		)
	}

	dependenciesValue, err := inputs.Get("dependencies")
	if err != nil {
		return test{}, fmt.Errorf(
			"Missing required argument 'dependencies'",
		)
	}
	dependencies, ok := dependenciesValue.(core.ArtifactID)
	if !ok {
		return test{}, fmt.Errorf(
			"'dependencies' argument must be a py_virtualenv target",
		)
	}

	sourcesValue, err := inputs.Get("sources")
	if err != nil {
		return test{}, fmt.Errorf(
			"Missing required argument 'sources'",
		)
	}
	sources, ok := sourcesValue.(core.ArtifactID)
	if !ok {
		return test{}, fmt.Errorf(
			"'sources' argument must be a filegroup; got %T",
			sourcesValue,
		)
	}

	return test{
		directory:    string(directory),
		dependencies: dependencies,
		sources:      sources,
	}, nil
}

func testBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	test, err := testParseInputs(dag.Inputs)
	if err != nil {
		return err
	}
	return testRun(dag, cache, stdout, stderr, test)
}

func testRun(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	test test,
) error {
	if _, err := cache.TempDir(
		func(dir string) (string, core.ArtifactID, error) {
			outputRelPath := "output"
			outputFilePath := filepath.Join(dir, outputRelPath)

			// closure b/c of defer outputFile.Close() below
			err := func() error {
				outputFile, err := os.Create(outputFilePath)
				if err != nil {
					return errors.Wrap(err, "Opening output file")
				}
				defer outputFile.Close()

				venvBinDir := filepath.Join(
					cache.Path(test.dependencies),
					"bin",
				)
				// Run the `python` from the virtualenv directory. This should
				// be sufficient to run this in the virtualenv, but we're also
				// updating the PATH environment variable to include the
				// `venvBinDir` as well.
				cmd := exec.Command(
					filepath.Join(venvBinDir, "python"),
					"-m",
					"pytest",
				)
				cmd.Stdout = io.MultiWriter(stdout, outputFile)
				cmd.Stderr = stderr
				cmd.Dir = filepath.Join(
					cache.Path(test.sources),
					test.directory,
				)
				cmd.Env = prependPATH(venvBinDir)
				if err := cmd.Run(); err != nil {
					return errors.Wrapf(err, "Running pytest")
				}
				return nil
			}()
			return outputRelPath, dag.ID.ArtifactID(), err
		},
	); err != nil {
		return errors.Wrap(err, "Running Python tests")
	}
	return nil
}

var Test = core.Plugin{Type: BuilderTypeTest, BuildScript: testBuildScript}
