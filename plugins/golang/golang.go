package golang

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"io/ioutil"

	"github.com/weberc2/builder/core"
)

type goBinaryInputs struct {
	packageName string
	modfile     core.ArtifactID
	sumfile     core.ArtifactID
	sources     []core.ArtifactID
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

	modFileValue, err := inputs.Get("modfile")
	if err != nil {
		return goBinaryInputs{}, err
	}
	modFile, ok := modFileValue.(core.ArtifactID)
	if !ok {
		return goBinaryInputs{}, fmt.Errorf(
			"TypeError: Invalid type for modfile argument; "+
				"expected artifact ID, found %T",
			modFileValue,
		)
	}

	sumFileValue, err := inputs.Get("sumfile")
	if err != nil {
		return goBinaryInputs{}, err
	}
	sumFile, ok := sumFileValue.(core.ArtifactID)
	if !ok {
		return goBinaryInputs{}, fmt.Errorf(
			"TypeError: Invalid type for sumfile argument; "+
				"expected artifact ID, found %T",
			sumFileValue,
		)
	}

	sourcesValue, err := inputs.Get("sources")
	if err != nil {
		return goBinaryInputs{}, fmt.Errorf(
			"Missing required argument 'sources'",
		)
	}
	sourcesArray, ok := sourcesValue.(core.FrozenArray)
	if !ok {
		return goBinaryInputs{}, fmt.Errorf(
			"'sources' argument must be a list",
		)
	}
	if len(sourcesArray) < 1 {
		return goBinaryInputs{}, fmt.Errorf(
			"'sources' must contain at least one source",
		)
	}
	sources := make([]core.ArtifactID, len(sourcesArray))
	for i, sourceValue := range sourcesArray {
		if source, ok := sourceValue.(core.ArtifactID); ok {
			sources[i] = source
			continue
		}
		return goBinaryInputs{}, fmt.Errorf(
			"'sources' elements must be artifact IDs; found %T at index %d",
			sourceValue,
			i,
		)
	}
	return goBinaryInputs{
		packageName: string(packageName),
		modfile:     modFile,
		sumfile:     sumFile,
		sources:     sources,
	}, nil
}

func loadFromCache(
	artifactID core.ArtifactID,
	cache core.Cache,
	workspace string,
) error {
	filePath := filepath.Join(workspace, artifactID.FilePath)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return cache.Read(artifactID, func(r io.Reader) error {
		_, err := io.Copy(file, r)
		return err
	})
}

func loadGoBinaryInputs(
	inputs goBinaryInputs,
	cache core.Cache,
	workspace string,
) error {
	if err := loadFromCache(inputs.modfile, cache, workspace); err != nil {
		return err
	}
	if err := loadFromCache(inputs.sumfile, cache, workspace); err != nil {
		return err
	}
	for _, source := range inputs.sources {
		if err := loadFromCache(source, cache, workspace); err != nil {
			return err
		}
	}
	return nil
}

func goBinaryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.Remove(tempdir)

	goBinInputs, err := parseGoBinaryInputs(dag.Inputs)
	if err != nil {
		return err
	}

	workspace := filepath.Join(tempdir, "src", goBinInputs.packageName)
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return err
	}

	if err := loadGoBinaryInputs(goBinInputs, cache, workspace); err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-o", cache.Path(dag.ID.ArtifactID()))
	cmd.Dir = workspace
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

var Binary = core.Plugin{
	Type: "go_binary",
	Factory: func(args core.FrozenObject) (core.BuildScript, error) {
		return goBinaryBuildScript, nil
	},
}
