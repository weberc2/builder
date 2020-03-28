package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/weberc2/builder/core"
	"github.com/weberc2/builder/plugins/command"
	"github.com/weberc2/builder/plugins/git"
	"github.com/weberc2/builder/plugins/golang"
	"github.com/weberc2/builder/plugins/python"
	"go.starlark.net/starlark"
)

type workspace struct {
	root string
	id   string
}

func findRoot(start string) (workspace, error) {
	entries, err := ioutil.ReadDir(start)
	if err != nil {
		return workspace{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() == "WORKSPACE" {
			data, err := ioutil.ReadFile(filepath.Join(start, entry.Name()))
			if err != nil {
				return workspace{}, err
			}
			if id := strings.TrimSpace(string(data)); len(id) > 0 {
				return workspace{root: start, id: id}, nil
			}
			return workspace{}, errors.New(
				"WORKSPACE must contain workspace ID",
			)
		}
	}
	if start == "/" {
		return workspace{}, fmt.Errorf("WORKSPACE not found")
	}
	return findRoot(filepath.Dir(start))
}

var plugins = []core.Plugin{
	git.Clone,
	command.Command,

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

func build(ctx *cli.Context, cache core.Cache, dag core.DAG) error {
	return core.Build(core.LocalExecutor(plugins, cache), dag)
}

func run(ctx *cli.Context, cache core.Cache, dag core.DAG) error {
	if err := build(ctx, cache, dag); err != nil {
		return err
	}

	cmd := exec.Command(cache.Path(dag.ID.ArtifactID()), ctx.Args()[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func graph(dag core.DAG, indent string) {
	fmt.Printf("%s%s", indent, dag.ID)
	if len(dag.Dependencies) > 0 {
		fmt.Println(" {")
		for _, dependency := range dag.Dependencies {
			graph(dependency, indent+"  ")
			fmt.Println(",")
		}
		fmt.Print(indent + "}")
	}
}

func targetAction(
	f func(ctx *cli.Context, t *core.Target, workspace workspace) error,
) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return errors.New("Missing PACKAGE:TARGET argument")
		}

		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		workspace, err := findRoot(pwd)
		if err != nil {
			return err
		}

		targetID, err := core.ParseTargetID(workspace.root, pwd, ctx.Args()[0])
		if err != nil {
			return errors.Errorf("Failed to parse target ID: %v", err)
		}

		targets, err := core.Evaluate(
			targetID.Package,
			workspace.root,
			map[string]string{
				"std/python":  python.BuiltinModule,
				"std/command": command.BuiltinModule,
				"std/golang":  golang.BuiltinModule,
				"std/git":     git.BuiltinModule,
			},
		)

		if err != nil {
			if evalErr, ok := errors.Cause(err).(*starlark.EvalError); ok {
				return errors.New(evalErr.Backtrace())
			}
			return errors.Errorf("Evaluation error: %v", err)
		}

		for i, target := range targets {
			if target.ID == targetID {
				return f(ctx, &targets[i], workspace)
			}
		}
		return errors.Errorf("Couldn't find target %s", targetID)
	}
}

func dagAction(
	f func(ctx *cli.Context, cache core.Cache, dag core.DAG) error,
) cli.ActionFunc {
	return targetAction(func(
		ctx *cli.Context,
		t *core.Target,
		workspace workspace,
	) error {
		cacheDir := "/tmp/cache"
		if home := os.Getenv("HOME"); home != "" {
			cacheDir = filepath.Join(home, ".cache/builder")
		}
		cache := core.LocalCache(workspace.id, cacheDir)

		dag, err := core.FreezeTarget(workspace.root, cache, *t)
		if err != nil {
			if evalErr, ok := err.(*starlark.EvalError); ok {
				return errors.New(evalErr.Backtrace())
			}
			return err
		}
		return f(ctx, cache, dag)
	})
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		cli.Command{
			Name:        "build",
			Usage:       "Build a target",
			UsageText:   "Build a target",
			Description: "Build a target",
			ArgsUsage: "Takes a single argument in the format " +
				"'PACKAGE:TARGET'",
			Action: dagAction(build),
		},
		cli.Command{
			Name:      "show",
			Aliases:   []string{"eval", "json"},
			Usage:     "Print JSON representation for a given target",
			UsageText: "Print JSON representation for a given target",
			Description: "Takes a target identifier (PACKAGE:TARGET), finds " +
				"the corresponding target definition (starlark), evaluates " +
				"it into a target, and renders the target as JSON.",
			ArgsUsage: "Takes a single argument in the format " +
				"'PACKAGE:TARGET'",
			Action: targetAction(func(
				ctx *cli.Context,
				t *core.Target,
				workspace workspace,
			) error {
				data, err := json.MarshalIndent(t, "", "    ")
				if err != nil {
					return errors.Wrapf(
						err,
						"Failed to marshal target %s",
						t.ID,
					)
				}
				fmt.Printf("%s\n", data)
				return nil
			}),
		},
		cli.Command{
			Name:      "checksum",
			Aliases:   []string{"fingerprint"},
			Usage:     "Print the checksum of a target",
			UsageText: "Print the checksum for a target",
			ArgsUsage: "Takes a single argument in the format " +
				"'PACKAGE:TARGET'",
			Action: dagAction(func(
				ctx *cli.Context,
				cache core.Cache,
				dag core.DAG,
			) error {
				_, err := fmt.Println(dag.ID.Checksum)
				return err
			}),
		},
		cli.Command{
			Name:    "cache-path",
			Aliases: []string{"path"},
			Usage:   "Print the cache path for a target at the current version",
			UsageText: "Print the cache path for a target at the current " +
				"version",
			Description: "For a target ID (PACKAGE:TARGET), this command " +
				"evaluates and fully hashes the target to determine where " +
				"the final cache path for the artifact. This does not build " +
				"the artifact nor does it depend on the artifact having " +
				"been built previously at the current version.",
			ArgsUsage: "Takes a single argument in the format " +
				"'PACKAGE:TARGET'",
			Action: dagAction(func(
				ctx *cli.Context,
				cache core.Cache,
				dag core.DAG,
			) error {
				_, err := fmt.Println(cache.Path(dag.ID.ArtifactID()))
				return err
			}),
		},
		cli.Command{
			Name:      "run",
			Usage:     "Tries to execute a build artifact",
			UsageText: "Tries to execute a build artifact",
			Description: fmt.Sprintf(
				"Builds the target artifact and attempts to execute it (via "+
					"subprocess). This is conceptually the same as `%s build "+
					"PACKAGE:TARGET && $(%s path PACKAGE:TARGET)`.",
				os.Args[0],
				os.Args[0],
			),
			ArgsUsage: "Takes a single argument in the format " +
				"'PACKAGE:TARGET'",
			Action: dagAction(run),
		},
		cli.Command{
			Name:        "graph",
			Usage:       "Graphs the dependencies",
			UsageText:   "Graphs the dependencies",
			Description: "Render the dependency graph as plaintext",
			ArgsUsage: "Takes a single argument in the format " +
				"'PACKAGE:TARGET'",
			Action: dagAction(func(
				_ *cli.Context,
				_ core.Cache,
				dag core.DAG,
			) error {
				graph(dag, "")
				fmt.Println()
				return nil
			}),
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
