package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type BuilderType string

type BuildScript func(inputs FrozenObject, w io.Writer) error

type Plugin struct {
	Type    BuilderType
	Factory func(args FrozenObject) (BuildScript, error)
}

type DAG struct {
	FrozenTarget
	Dependencies []DAG
}

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

func main() {
	if len(os.Args) < 2 {
		log.Fatal("USAGE: builder <target>")
	}
	targetID, err := parseTargetID(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	cache := LocalCache{"/tmp/cache"}
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	root, err := findRoot(pwd)
	if err != nil {
		log.Fatal(err)
	}

	f := freezer{
		root:      root,
		cache:     cache,
		evaluator: Evaluator{root},
	}

	dag, err := f.freezeTargetID(targetID)
	if err != nil {
		log.Fatal(err)
	}

	if err := Build(
		LocalExecutor(
			[]Plugin{{
				Type: "noop",
				Factory: func(args FrozenObject) (BuildScript, error) {
					return func(inputs FrozenObject, w io.Writer) error {
						_, err := fmt.Fprintln(w, "Done!")
						return err
					}, nil
				},
			}, {
				Type: "golang",
				Factory: func(args FrozenObject) (BuildScript, error) {
					return func(inputs FrozenObject, w io.Writer) error {
						filePath := filepath.Join(os.TempDir(), "output")
						cmd := exec.Command("go", "build", "-o", filePath)
						if err := cmd.Run(); err != nil {
							return err
						}
						f, err := os.Open(filePath)
						if err != nil {
							return err
						}
						defer f.Close()

						_, err = io.Copy(w, f)
						return err
					}, nil
				},
			}},
			cache,
		),
		dag,
	); err != nil {
		log.Fatal(err)
	}
}
