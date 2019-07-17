package core

import (
	"errors"
	"log"
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
							log.Printf("Found artifact %s", dag.ID.ArtifactID())
						}
						return err
					}
					log.Printf("Missing artifact %s", dag.ID.ArtifactID())
				}
				log.Printf("Building %s", dag.ID.ArtifactID())

				buildScript, err := plugin.Factory(dag.BuilderArgs)
				if err != nil {
					return err
				}

				return buildScript(
					dag.Inputs,
					dag.ID.ArtifactID(),
					cache,
					dag.Dependencies,
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

	return execute(dag)
}
