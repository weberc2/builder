package golang

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/weberc2/builder/core"
)

type goBinaryInputs struct {
	packageName string
	sources     core.ArtifactID
}

func parseGoBinaryInputs(inputs core.FrozenObject) (goBinaryInputs, error) {
	packageNameValue, err := inputs.Get("package_name")
	if err != nil {
		return goBinaryInputs{}, err
	}
	packageName, ok := packageNameValue.(core.String)
	if !ok {
		return goBinaryInputs{}, fmt.Errorf(
			"TypeError: package_name argument must be string; got %T",
			packageNameValue,
		)
	}

	sourcesValue, err := inputs.Get("sources")
	if err != nil {
		return goBinaryInputs{}, fmt.Errorf(
			"Missing required argument 'sources'",
		)
	}
	sources, ok := sourcesValue.(core.ArtifactID)
	if !ok {
		return goBinaryInputs{}, fmt.Errorf(
			"'sources' argument must be a filegroup",
		)
	}

	return goBinaryInputs{
		packageName: string(packageName),
		sources:     sources,
	}, nil
}

func goBinaryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	goBinInputs, err := parseGoBinaryInputs(dag.Inputs)
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-o", cache.Path(dag.ID.ArtifactID()))
	cmd.Dir = cache.Path(goBinInputs.sources)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "GOMODULE111=on")
	return cmd.Run()
}

var Binary = core.Plugin{
	Type: "go_binary",
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return goBinaryBuildScript, nil
	},
}
