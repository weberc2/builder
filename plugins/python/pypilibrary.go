package python

import (
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

type pypiLibrary struct {
	packageName  string
	constraint   string
	dependencies []core.ArtifactID
}

func (pl *pypiLibrary) parseInputs(inputs core.FrozenObject) error {
	return errors.Wrap(
		inputs.VisitKeys(
			core.KeySpec{
				Key:   "package_name",
				Value: core.ParseString(&pl.packageName),
			},
			core.KeySpec{
				Key:   "constraint",
				Value: core.ParseString(&pl.constraint),
			},
			core.KeySpec{
				Key: "dependencies",
				Value: core.AssertArrayOf(
					core.AssertArtifactID(
						func(dep core.ArtifactID) error {
							pl.dependencies = append(pl.dependencies, dep)
							return nil
						},
					),
				),
			},
		),
		"Parsing pypi_library inputs",
	)
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
		lib.packageName+lib.constraint,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if err := os.RemoveAll(cache.Path(output)); err != nil {
			if !os.IsNotExist(err) {
				log.Printf(
					"ERROR: Failed to delete directory '%s' from cache. "+
						"This will give the appearance that the artifact was"+
						"created when in fact it was not. Please make sure "+
						"this is deleted before proceeding.",
					cache.Path(output),
				)
			}
			// If the error is "directory doesn't exist", then that's fine too
		}
		return errors.Wrap(err, "Installing pypi library")
	}
	return nil
}

func pypiLibraryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	var lib pypiLibrary
	if err := lib.parseInputs(dag.Inputs); err != nil {
		return err
	}

	return pypiLibraryInstall(dag.ID.ArtifactID(), cache, stdout, stderr, lib)
}

var PypiLibrary = core.Plugin{
	Type:        BuilderTypePypiLibrary,
	BuildScript: pypiLibraryBuildScript,
}
