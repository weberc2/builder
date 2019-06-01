package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/adler32"
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

type Cache interface {
	Write(id ArtifactID, f func(io.Writer) error) error
	Exists(id ArtifactID) error
}

var ErrPluginNotFound = errors.New("Plugin not found")

type DAG struct {
	FrozenTarget
	Dependencies []DAG
}

func visitArtifactID(input FrozenInput, f func(ArtifactID) error) error {
	switch x := input.(type) {
	case FrozenObject:
		for _, field := range x {
			if err := visitArtifactID(field.Value, f); err != nil {
				return err
			}
		}
	case FrozenArray:
		for _, elt := range x {
			if err := visitArtifactID(elt, f); err != nil {
				return err
			}
		}
	case ArtifactID:
		return f(x)
	}
	return nil
}

func Build(execute ExecuteFunc, dag DAG) error {
	for _, dependency := range dag.Dependencies {
		if err := Build(execute, dependency); err != nil {
			return err
		}
	}

	return execute(dag.FrozenTarget)
}

func ChecksumBytes(bs []byte) uint32 { return adler32.Checksum(bs) }

func ChecksumString(s string) uint32 { return ChecksumBytes([]byte(s)) }

func JoinChecksums(checksums ...uint32) uint32 {
	buf := make([]byte, len(checksums)*4)
	for i, checksum := range checksums {
		binary.BigEndian.PutUint32(buf[i*4:i*4+4], checksum)
	}
	return ChecksumBytes(buf)
}

type LocalCache struct {
	Directory string
}

func (lc LocalCache) Write(id ArtifactID, f func(w io.Writer) error) error {
	artifactPath := filepath.Join(
		lc.Directory,
		string(id.Package),
		string(id.Target),
		id.FilePath,
		fmt.Sprint(id.Checksum),
	)

	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(artifactPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return f(file)
}

var ErrArtifactNotFound = errors.New("Artifact not found")

func (lc LocalCache) Exists(id ArtifactID) error {
	_, err := os.Stat(filepath.Join(
		lc.Directory,
		string(id.Package),
		string(id.Target),
		id.FilePath,
		fmt.Sprint(id.Checksum),
	))
	if os.IsNotExist(err) {
		return ErrArtifactNotFound
	}
	return err
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
