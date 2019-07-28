package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/weberc2/builder/core"
	"github.com/weberc2/builder/plugins"
	"go.starlark.net/starlark"
)

func findRoot(start string) (string, error) {
	entries, err := ioutil.ReadDir(start)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() == "WORKSPACE" {
			return start, nil
		}
	}
	if start == "/" {
		return "", fmt.Errorf("WORKSPACE not found")
	}
	return findRoot(filepath.Dir(start))
}

func build(cache core.Cache, dag core.DAG) error {
	return _build(cache, dag, false)
}

func rebuild(cache core.Cache, dag core.DAG) error {
	return _build(cache, dag, true)
}

func _build(cache core.Cache, dag core.DAG, rebuild bool) error {
	return core.Build(
		core.LocalExecutor(
			[]core.Plugin{
				plugins.GitClone,
				plugins.GoBinary,
				plugins.GoLibrary,
				plugins.PySourceBinary,
				plugins.PySourceLibrary,
				plugins.PyPypiLibrary,
				plugins.PyTest,
				plugins.PyVirtualEnv,
			},
			cache,
			rebuild,
		),
		dag,
	)
}

func run(cache core.Cache, dag core.DAG) error {
	if err := build(cache, dag); err != nil {
		return err
	}

	cmd := exec.Command(cache.Path(dag.ID.ArtifactID()), os.Args[3:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func main() {
	var command func(core.Cache, core.DAG) error
	if len(os.Args) > 2 {
		switch os.Args[1] {
		case "build":
			command = build
		case "rebuild":
			command = rebuild
		case "run":
			command = run
		}
	}

	if command == nil {
		log.Fatal("USAGE: builder <build|run> <target>")
	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	root, err := findRoot(pwd)
	if err != nil {
		log.Fatal(err)
	}

	targetID, err := core.ParseTargetID(root, pwd, os.Args[2])
	if err != nil {
		log.Fatalf("Failed to parse target ID: %v", err)
	}

	cache := core.LocalCache("/tmp/cache")

	dag, err := core.FreezeTargetID(
		root,
		cache,
		core.Evaluator{root},
		targetID,
	)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		}
		log.Fatal(err)
	}

	if err := command(cache, dag); err != nil {
		log.Fatal(err)
	}
}
