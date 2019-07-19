package core

import (
	"bytes"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

var ErrPluginNotFound = errors.New("Plugin not found")

type ExecuteFunc func(dag DAG) error

func LocalExecutor(plugins []Plugin, cache Cache, rebuild bool) ExecuteFunc {
	return func(dag DAG) error {
		for _, plugin := range plugins {
			if plugin.Type == dag.BuilderType {
				if !rebuild {
					if err := cache.Exists(
						dag.ID.ArtifactID(),
					); err != ErrArtifactNotFound {
						if err == nil {
							color.Green("Found artifact %s", dag.ID.ArtifactID())
						}
						return err
					}
				}
				color.Yellow("Building %s", dag.ID.ArtifactID())

				buildScript, err := plugin.Factory(dag.BuilderArgs)
				if err != nil {
					return err
				}

				var stdout, stderr bytes.Buffer
				if err := buildScript(
					dag,
					cache,
					&stdout,
					&stderr,
				); err != nil {
					// If the build script failed, copy the build script's
					// stdout and stderr to system stderr
					if handleErr := func() error {
						if _, err := io.Copy(os.Stderr, &stdout); err != nil {
							return err
						}

						if _, err := color.New(color.FgRed).Fprintln(
							os.Stderr,
							stderr.String(),
						); err != nil {
							return err
						}

						return nil
					}(); handleErr != nil {
						err = errors.Wrapf(handleErr, "Handling '%v'", err)
					}

					return errors.Wrapf(
						err,
						"Building target %s",
						dag.ID.ArtifactID(),
					)
				}

				return nil
			}
		}

		return errors.Wrapf(ErrPluginNotFound, "plugin = %s", dag.BuilderType)
	}
}

func Build(execute ExecuteFunc, dag DAG) error {
	for _, dependency := range dag.Dependencies {
		if err := Build(execute, dependency); err != nil {
			return err
		}
	}

	return execute(dag)
}
