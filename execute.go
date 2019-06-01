package main

import (
	"errors"
	"io"
	"log"
)

var ErrPluginNotFound = errors.New("Plugin not found")

type ExecuteFunc func(FrozenTarget) error

func LocalExecutor(plugins []Plugin, cache Cache) ExecuteFunc {
	return func(ft FrozenTarget) error {
		for _, plugin := range plugins {
			if plugin.Type == ft.BuilderType {
				if err := cache.Exists(
					ft.ID.ArtifactID(),
				); err != ErrArtifactNotFound {
					if err == nil {
						log.Printf("Found artifact %s", ft.ID.ArtifactID())
					}
					return err
				}
				log.Printf("Missing artifact %s", ft.ID.ArtifactID())

				buildScript, err := plugin.Factory(ft.BuilderArgs)
				if err != nil {
					return err
				}

				return cache.Write(
					ft.ID.ArtifactID(),
					func(w io.Writer) error {
						log.Printf("INFO Building %s", ft.ID)
						return buildScript(ft.Inputs, w)
					},
				)
			}
		}

		return ErrPluginNotFound
	}
}

func Build(execute ExecuteFunc, dag DAG) error {
	for _, dependency := range dag.Dependencies {
		if err := Build(execute, dependency); err != nil {
			return err
		}
	}

	return execute(dag.FrozenTarget)
}
