package python

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

func testBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	var directory string
	var dependencies core.ArtifactID
	var sources core.ArtifactID
	if err := dag.Inputs.VisitKeys(
		core.KeySpec{
			Key:   "directory",
			Value: core.ParseString(&directory),
		},
		core.KeySpec{
			Key: "dependencies",
			Value: core.AssertArtifactID(
				func(virtualenv core.ArtifactID) error {
					dependencies = virtualenv
					return nil
				},
			),
		},
		core.KeySpec{
			Key:   "sources",
			Value: core.ParseArtifactID(&sources),
		},
	); err != nil {
		return errors.Wrap(err, "Parsing py_test inputs")
	}
	return testRun(
		dag,
		cache,
		stdout,
		stderr,
		directory,
		dependencies,
		sources,
	)
}

func testRun(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	directory string,
	dependencies core.ArtifactID,
	sources core.ArtifactID,
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

				if err := venvCmd(
					cache,
					dependencies,
					command{
						Command: "pytest",
						Dir:     filepath.Join(cache.Path(sources), directory),
						Stdout:  io.MultiWriter(stdout, outputFile),
						Stderr:  stderr,
						Env:     os.Environ(),
					},
				).Run(); err != nil {
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
