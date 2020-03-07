package command

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/buildutil"
	"github.com/weberc2/builder/core"
)

var Command = core.Plugin{
	Type: core.BuilderType("command"),
	BuildScript: func(
		dag core.DAG,
		cache core.Cache,
		stdout io.Writer,
		stderr io.Writer,
	) error {
		var command string
		var args []string
		var environment []string
		if err := dag.Inputs.VisitKeys(
			core.KeySpec{
				Key: "command",
				Value: core.Match(
					core.ParseString(&command),
					core.AssertArtifactID(func(id core.ArtifactID) error {
						command = cache.Path(id)
						return nil
					}),
				),
			},
			core.KeySpec{
				Key: "args",
				Value: core.AssertArrayOf(core.Match(
					core.AssertString(func(s string) error {
						args = append(args, s)
						return nil
					}),
					core.AssertArtifactID(func(id core.ArtifactID) error {
						args = append(args, cache.Path(id))
						return nil
					}),
				)),
			},
			core.KeySpec{
				Key: "environment",
				Value: core.AssertObjectOf(func(field core.FrozenField) error {
					var s string
					switch x := field.Value.(type) {
					case core.String:
						s = string(x)
					case core.ArtifactID:
						s = cache.Path(x)
					default:
						return core.NewTypeErr(
							"Union[str, Target]",
							field.Value,
						)
					}
					environment = append(
						environment,
						fmt.Sprintf("%s=%s", field.Key, s),
					)
					return nil
				}),
			},
		); err != nil {
			return errors.Wrap(err, "Running command() build script")
		}

		return buildutil.Build(
			dag,
			cache,
			stdout,
			stderr,
			func(ctx *buildutil.BuildContext) error {
				environment = append(
					environment,
					fmt.Sprintf("OUTPUT=%s", ctx.Output),
				)
				return ctx.Call(
					command,
					ctx.Workspace,
					append(os.Environ(), environment...),
					args...,
				)
			},
		)
	},
}

const BuiltinModule = `
def bash(name, script, environment = None):
    return mktarget(
        name = name,
        type = "command",
        args = {
            "command": "bash",
            "environment": environment if environment != None else {},
            "args": [ "-c", "set -e\n{}".format(script) ],
        },
    )
`
