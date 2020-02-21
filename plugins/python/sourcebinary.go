package python

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

type sourceBinary struct {
	packageName  string
	entryPoint   string
	dependencies []core.ArtifactID
	sources      core.ArtifactID
	pexVenv      core.ArtifactID
}

func (sb *sourceBinary) parseInputs(inputs core.FrozenObject) error {
	return errors.Wrap(
		inputs.VisitKeys(
			core.KeySpec{
				Key:   "package_name",
				Value: core.ParseString(&sb.packageName),
			},
			core.KeySpec{
				Key:   "entry_point",
				Value: core.ParseString(&sb.entryPoint),
			},
			core.KeySpec{
				Key: "dependencies",
				Value: core.AssertArrayOf(core.AssertArtifactID(
					func(dep core.ArtifactID) error {
						sb.dependencies = append(sb.dependencies, dep)
						return nil
					},
				)),
			},
			core.KeySpec{
				Key:   "sources",
				Value: core.ParseArtifactID(&sb.sources),
			},
			core.KeySpec{
				Key:   "pex_venv",
				Value: core.ParseArtifactID(&sb.pexVenv),
			},
		),
		"Parsing py_source_binary inputs",
	)
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

	if err := venvCmd(
		cache,
		bin.pexVenv,
		command{
			Command: "pex",
			Args: append(
				append(
					[]string{
						"--disable-cache",
						"--python",
						"python3.6",
						"--no-index",
					},
					append(wheelPaths, wheelPath)...,
				),
				"-o",
				cache.Path(dag.ID.ArtifactID()),
				"-e",
				fmt.Sprintf("%s:%s", bin.packageName, bin.entryPoint),
			),
			Stdout: stdout,
			Stderr: stderr,
			Env:    os.Environ(),
		},
	).Run(); err != nil {
		return errors.Wrapf(
			err,
			"Building pex for target %s",
			dag.ID.ArtifactID(),
		)
	}

	return nil
}

func sourceBinaryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	var bin sourceBinary
	if err := bin.parseInputs(dag.Inputs); err != nil {
		return err
	}
	return sourceBinaryInstall(dag, cache, stdout, stderr, bin)
}

var SourceBinary = core.Plugin{
	Type:        BuilderTypeSourceBinary,
	BuildScript: sourceBinaryBuildScript,
}
