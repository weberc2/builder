package core

import (
	"bytes"
	"io"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/weberc2/builder/paralleltree"
)

type printer struct {
	stdout *os.File
	stderr *os.File
	l      sync.Mutex
}

func (p *printer) writeString(f *os.File, s string) {
	p.l.Lock()
	if _, err := f.WriteString(s + "\n"); err != nil {
		log.Printf("ERROR writing to file: %v", err)
	}
	p.l.Unlock()
}

func (p *printer) Success(format string, v ...interface{}) {
	p.writeString(p.stdout, color.GreenString(format, v...))
}

func (p *printer) Error(format string, v ...interface{}) {
	p.writeString(p.stderr, color.RedString(format, v...))
}

func (p *printer) Info(format string, v ...interface{}) {
	p.writeString(p.stdout, color.YellowString(format, v...))
}

var ErrPluginNotFound = errors.New("Plugin not found")

type ExecuteFunc func(dag DAG) error

func LocalExecutor(plugins []Plugin, cache Cache) ExecuteFunc {
	printer := &printer{stdout: os.Stdout, stderr: os.Stderr}
	return func(dag DAG) error {
		for _, plugin := range plugins {
			if plugin.Type == dag.BuilderType {
				if err := cache.Exists(
					dag.ID.ArtifactID(),
				); err != ErrArtifactNotFound {
					if err == nil {
						printer.Success(
							"Found artifact %s",
							dag.ID.ArtifactID(),
						)
					}
					return err
				}
				printer.Info("Building %s", dag.ID.ArtifactID())

				var stdout, stderr bytes.Buffer
				if err := plugin.BuildScript(
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

						printer.Error(stderr.String())

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

// func Build(execute ExecuteFunc, dag DAG) error {
// 	return build(execute, dag, map[FrozenTargetID]struct{}{})
// }

func build(
	execute ExecuteFunc,
	dag DAG,
	seen *frozenTargetSet,
) error {
	for _, dependency := range dag.Dependencies {
		if seen.exists(dependency.ID) {
			continue
		}

		if err := build(execute, dependency, seen); err != nil {
			return err
		}
		seen.put(dependency.ID)
	}

	return execute(dag)
}

type frozenTargetSet struct {
	m map[FrozenTargetID]struct{}
	l sync.RWMutex
}

func (fts *frozenTargetSet) put(ftid FrozenTargetID) {
	fts.l.Lock()
	fts.m[ftid] = struct{}{}
	fts.l.Unlock()
}

func (fts *frozenTargetSet) exists(ftid FrozenTargetID) bool {
	fts.l.RLock()
	_, found := fts.m[ftid]
	fts.l.RUnlock()
	return found
}

func dagToNode(
	execute ExecuteFunc,
	dag DAG,
	seen *frozenTargetSet,
	seen2 map[string]struct{},
) *paralleltree.Node {
	var children []*paralleltree.Node
	for _, dependency := range dag.Dependencies {
		if _, found := seen2[dependency.ID.String()]; found {
			continue
		}
		children = append(children, dagToNode(execute, dependency, seen, seen2))
		seen2[dependency.ID.String()] = struct{}{}
	}

	return paralleltree.NewNode(
		dag.ID.String(),
		children,
		func() error {
			if seen.exists(dag.ID) {
				return nil
			}
			return execute(dag)
		},
	)
}

func Build(execute ExecuteFunc, dag DAG) error {
	return paralleltree.ProcessConcurrently(
		dagToNode(
			execute,
			dag,
			&frozenTargetSet{m: map[FrozenTargetID]struct{}{}},
			map[string]struct{}{},
		),
		2*runtime.NumCPU(),
	)
}
