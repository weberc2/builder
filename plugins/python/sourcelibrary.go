package python

import (
	"io"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

var SourceLibrary = core.Plugin{
	Type: BuilderTypeSourceLibrary,
	BuildScript: func(
		dag core.DAG,
		cache core.Cache,
		stdout io.Writer,
		stderr io.Writer,
	) error {
		return errors.Wrapf(
			dag.Inputs.VisitKeys(
				core.KeySpec{
					Key: "package_name",
					Value: core.AssertString(
						func(string) error { return nil },
					),
				},
				core.KeySpec{
					Key: "dependencies",
					Value: core.AssertArrayOf(core.AssertArtifactID(
						func(core.ArtifactID) error { return nil },
					)),
				},
				core.KeySpec{
					Key: "sources",
					Value: core.AssertArtifactID(
						func(sources core.ArtifactID) error {
							return buildWheel(
								cache.Path(sources),
								cache.Path(dag.ID.ArtifactID()),
								stdout,
								stderr,
							)
						},
					),
				},
			),
			"Parsing python_library inputs for target %s",
			dag.ID,
		)
	},
}
