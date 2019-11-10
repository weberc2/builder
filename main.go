package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/weberc2/builder/core"
	"github.com/weberc2/builder/plugins/git"
	"github.com/weberc2/builder/plugins/golang"
	"github.com/weberc2/builder/plugins/python"
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

var plugins = []core.Plugin{
	git.Clone,
	golang.Library,
	golang.Binary,
	python.SourceBinary,
	python.SourceLibrary,
	python.PypiLibrary,
	python.Test,
	python.VirtualEnv,

	// Create a noop plugin. This is useful for meta-packages.
	core.Plugin{
		Type: core.BuilderType("noop"),
		BuildScript: func(
			dag core.DAG,
			cache core.Cache,
			stdout io.Writer,
			stderr io.Writer,
		) error {
			return cache.Write(
				dag.ID.ArtifactID(),
				func(w io.Writer) error {
					_, err := w.Write([]byte("noop"))
					return err
				},
			)
		},
	},
}

func build(cache core.Cache, dag core.DAG) error {
	return core.Build(core.LocalExecutor(plugins, cache), dag)
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

func graph(dag core.DAG) {
	for _, dependency := range dag.Dependencies {
		graph(dependency)
	}

	fmt.Printf("%s:\n", dag.ID)
	for _, dependency := range dag.Dependencies {
		fmt.Printf("  %s\n", dependency.ID)
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("USAGE: builder <command> <target>")
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

	var t *core.Target
	targets, err := core.Evaluator{
		BuiltinModules: map[string]string{
			"std/python": python.BuiltinModule,
		},
		PackageRoot: root,
	}.Evaluate(targetID.Package)
	if err != nil {
		log.Fatalf("Evaluation error: %v", err)
	}
	for i, target := range targets {
		if target.ID == targetID {
			t = &targets[i]
		}
	}
	if t == nil {
		log.Fatalf("Couldn't find target %s", targetID)
	}

	var command func(core.Cache, core.DAG) error
	switch os.Args[1] {
	case "show", "eval":
		data, err := json.MarshalIndent(t, "", "    ")
		if err != nil {
			log.Fatalf("Failed to marshal target %s: %v", targetID, err)
		}
		fmt.Printf("%s\n", data)
		os.Exit(0)
	case "cache-path", "path":
		command = func(cache core.Cache, dag core.DAG) error {
			fmt.Println(cache.Path(dag.ID.ArtifactID()))
			return nil
		}
	case "build":
		command = build
	case "run":
		command = run
	case "graph":
		command = func(_ core.Cache, dag core.DAG) error {
			graph(dag)
			return nil
		}
	default:
		log.Fatalf("Invalid command: %s", os.Args[1])
	}

	cache := core.LocalCache("/tmp/cache")

	dag, err := core.FreezeTarget(root, cache, *t)
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
