package python

import (
	"fmt"
	"io"
	"io/ioutil"
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
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating temp dir")
	}
	defer os.Remove(tmpDir)

	outputFilePath := filepath.Join(tmpDir, "output")
	if err := func() error { // closure b/c of defer outputFile.Close() below
		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			return errors.Wrap(err, "Opening output file")
		}
		defer outputFile.Close()

		cmd := exec.Command("pytest")
		cmd.Stdout = io.MultiWriter(stdout, outputFile)
		cmd.Stderr = stderr
		cmd.Dir = filepath.Join(cache.Path(test.sources), test.directory)
		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("PYTHONPATH=%s", cache.Path(test.dependencies)),
		)
		if err := cmd.Run(); err != nil {
			return errors.Wrapf(err, "Running pytest")
		}
		return nil
	}(); err != nil {
		return err
	}

	// Now that the tests have succeeded, copy the results into the cache
	if err := os.MkdirAll(
		filepath.Dir(cache.Path(dag.ID.ArtifactID())),
		0755,
	); err != nil {
		return errors.Wrap(err, "Creating parent directory in cache")
	}
	if err := os.Rename(
		outputFilePath,
		cache.Path(dag.ID.ArtifactID()),
	); err != nil {
		return errors.Wrap(err, "Moving test results from temp dir into cache")
	}
	return nil
}

var Test = core.Plugin{
	Type: BuilderTypeTest,
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return testBuildScript, nil
	},
}
