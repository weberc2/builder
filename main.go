package main

import (
	"errors"
	"fmt"
	"hash/adler32"
	"io"
	"log"
	"os"
	"path/filepath"
)

type Input interface{ input() }

type TargetName string

type PackageName string

type CacheRef struct {
	Package  PackageName
	Target   TargetName
	Checksum uint32
}

func (cr CacheRef) String() string {
	return fmt.Sprintf("%s:%s@%d", cr.Package, cr.Target, cr.Checksum)
}

func (cr CacheRef) input() {}

type Int int

func (i Int) input() {}

type String string

func (s String) input() {}

type Bool bool

func (b Bool) input() {}

type Field struct {
	Key   string
	Value Input
}

type Object []Field

func (o Object) input() {}

type Array []Input

func (a Array) input() {}

type Target struct {
	Name        TargetName
	Inputs      []Input
	BuildScript []byte
}

type BuilderType string

type BuildScript func(inputs Object, w io.Writer) error

type Plugin struct {
	Type    BuilderType
	Factory func(args Object) (BuildScript, error)
}

type FrozenTarget struct {
	ID          CacheRef
	Inputs      Object
	BuilderType BuilderType
	BuilderArgs Object
}

type ExecuteFunc func(FrozenTarget) error

func LocalExecutor(plugins []Plugin, cache Cache) ExecuteFunc {
	return func(ft FrozenTarget) error {
		for _, plugin := range plugins {
			if plugin.Type == ft.BuilderType {
				if err := cache.Exists(ft.ID); err != ErrArtifactNotFound {
					return err
				}

				buildScript, err := plugin.Factory(ft.BuilderArgs)
				if err != nil {
					return err
				}

				return cache.Write(ft.ID, func(w io.Writer) error {
					log.Printf("INFO Building %s", ft.ID)
					return buildScript(ft.Inputs, w)
				})
			}
		}

		return ErrPluginNotFound
	}
}

type Cache interface {
	Write(cr CacheRef, f func(io.Writer) error) error
	Exists(id CacheRef) error
}

var ErrPluginNotFound = errors.New("Plugin not found")

type DAG struct {
	FrozenTarget
	Dependencies []DAG
}

func visitCacheRef(input Input, f func(CacheRef) error) error {
	switch x := input.(type) {
	case Object:
		for _, field := range x {
			if err := visitCacheRef(field.Value, f); err != nil {
				return err
			}
		}
	case Array:
		for _, elt := range x {
			if err := visitCacheRef(elt, f); err != nil {
				return err
			}
		}
	case CacheRef:
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

type LocalCache struct {
	Directory string
}

func (lc LocalCache) Write(id CacheRef, f func(w io.Writer) error) error {
	dir := filepath.Join(lc.Directory, string(id.Package), string(id.Target))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(dir, fmt.Sprint(id.Checksum)))
	if err != nil {
		return err
	}
	defer file.Close()

	return f(file)
}

var ErrArtifactNotFound = errors.New("Artifact not found")

func (lc LocalCache) Exists(id CacheRef) error {
	_, err := os.Stat(filepath.Join(
		lc.Directory,
		string(id.Package),
		string(id.Target),
		fmt.Sprint(id.Checksum),
	))
	if os.IsNotExist(err) {
		return ErrArtifactNotFound
	}
	return err
}

func main() {
	dependency := CacheRef{
		Package:  "pkg",
		Target:   "dependency",
		Checksum: ChecksumString("456"),
	}

	if err := Build(
		LocalExecutor(
			[]Plugin{{
				Type: "noop",
				Factory: func(args Object) (BuildScript, error) {
					return func(inputs Object, w io.Writer) error {
						_, err := fmt.Fprintln(w, "Done!")
						return err
					}, nil
				},
			}},
			LocalCache{"/tmp/cache"},
		),
		DAG{
			FrozenTarget: FrozenTarget{
				ID: CacheRef{
					Package:  "pkg",
					Target:   "target",
					Checksum: ChecksumString("123"),
				},
				Inputs:      Object{Field{Key: "qux", Value: dependency}},
				BuilderType: "noop",
				BuilderArgs: nil,
			},
			Dependencies: []DAG{{FrozenTarget: FrozenTarget{
				ID:          dependency,
				Inputs:      nil,
				BuilderType: "noop",
				BuilderArgs: nil,
			}}},
		},
	); err != nil {
		log.Fatal(err)
	}
}
