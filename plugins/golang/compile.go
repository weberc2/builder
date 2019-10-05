package golang

import (
	"encoding/json"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

func jsonPrettyPrint(v interface{}) {
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", data)
}

func debugf(format string, vs ...interface{}) {
	// log.Println("DEBUG ", fmt.Sprintf(format, vs...))
}

func symlinkFiles2(dst, src string, srcFileInfo os.FileInfo) error {
	debugf("Symlinking files from %s into %s", src, dst)
	if err := os.MkdirAll(dst, srcFileInfo.Mode()); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		dstFilePath := filepath.Join(dst, file.Name())
		srcFilePath := filepath.Join(src, file.Name())
		if !file.IsDir() {
			if err := os.Symlink(srcFilePath, dstFilePath); err != nil {
				return err
			}
		}
	}

	return nil
}

func symlinkFiles(dst, src string, srcFileInfo os.FileInfo) error {
	debugf("Symlinking files from %s into %s", src, dst)
	if err := os.MkdirAll(dst, srcFileInfo.Mode()); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		dstFilePath := filepath.Join(dst, file.Name())
		srcFilePath := filepath.Join(src, file.Name())
		if file.IsDir() {
			if err := symlinkFiles(
				dstFilePath,
				srcFilePath,
				srcFileInfo,
			); err != nil {
				return err
			}
			continue
		}
		if err := os.Symlink(srcFilePath, dstFilePath); err != nil {
			return err
		}
	}

	return nil
}

func goInstall(
	gopath []string,
	packageName string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	cmd := exec.Command("go", "install", packageName)
	cmd.Env = append(
		os.Environ(),
		"GOPATH="+strings.Join(gopath, ":"),
		"GO111MODULE=off",
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	debugf("GOPATH %s", strings.Join(gopath, ":"))
	debugf("Running %s", strings.Join(cmd.Args, " "))
	fmt.Fprintf(stderr, "Running %s\n", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func buildLibrary(
	moduleName string,
	provides []string,
	sourcesDirectory string,
	dependencies []string,
	output string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	// Create a temporary workspace directory for this project. The workspace
	// directory will be structured like a GOPATH directory (it will have
	// 'src' and 'pkg' subdirs).
	workdir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating temporary working directory")
	}
	//defer os.Remove(workdir)

	// If the module doesn't explicitly enumerate any provided packages, it
	// indicates that the root directory is the provided package.
	if len(provides) < 1 {
		provides = append(provides, "")
	}

	for _, packageName := range provides {
		// We're symlinking the source files for each declared package into the
		// workdir for two reasons:
		//   1. Because we need them to compile each declared package and make
		//      sure it works.
		//   2. Because the workdir will get moved into the cache if all goes
		//      well, and we don't want modules that depend on this module to
		//      touch packages inside this module which this module didn't
		//      provide/declare/export (because those undeclared packages
		//      haven't been confirmed to build). The hypothetical
		//		dependant/downstream module would fail because the source code
		//      for the hypothetical undeclared package would not be available.
		//      It would fail anyway if the source code didn't compile (because
		//      the compiler would fail), but we want the failure to happen
		//      when we try to build the module that owns the broken package,
		//      not sometime downstream.
		log.Printf("Symlinking files for %s/%s", moduleName, packageName)
		// Symlink sources into the working directory
		packageSourcesDirectory := filepath.Join(sourcesDirectory, packageName)
		packageSourcesDirectoryInfo, err := os.Stat(packageSourcesDirectory)
		if err != nil {
			return errors.Wrapf(
				err,
				"stat()-ing the source directory in the cache: %s",
				sourcesDirectory,
			)
		}
		if err := symlinkFiles2(
			filepath.Join(workdir, "src", moduleName, packageName),
			packageSourcesDirectory,
			packageSourcesDirectoryInfo,
		); err != nil {
			return errors.Wrap(
				err,
				"Symlinking the source files into the tmp workspace",
			)
		}

		if err := goInstall(
			// prepend is necessary in cases where `packageName` depends on a
			// package defined in a nested directory. If the nested directory's
			// entry in the GOPATH is first, then the go toolchain will expect
			// to find `packageName`'s source code there when in fact it's in
			// `workdir`. E.g., `github.com/weberc2/builder/plugins` depends on
			// `github.com/weberc2/plugins/golang`.
			append([]string{workdir}, dependencies...),
			filepath.Join(moduleName, packageName),
			stdout,
			stderr,
		); err != nil {
			return errors.Wrapf(
				err,
				"Installing package %s/%s",
				moduleName,
				packageName,
			)
		}
	}

	// Create the output parent directory in the cache
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return errors.Wrapf(
			err,
			"Creating output parent directory in cache: %s",
			filepath.Dir(output),
		)
	}

	// Move the workspace into the cache; at this point the workspace is just a
	// directory with a `pkg` directory containing the compiled artifacts for
	// the module we created. It is structured this way so that other packages
	// that depend on this one can just append this cache directory to their
	// GOPATH--no need to copy or symlink artifacts.
	if err := os.Rename(workdir, output); err != nil {
		return errors.Wrapf(
			err,
			"Moving workspace into cache (%s -> %s)",
			workdir,
			output,
		)
	}
	return nil
}

func libraryBuildScript2(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	moduleName, err := dag.Inputs.GetString("module_name")
	if err != nil {
		return err
	}

	sourcesValue, err := dag.Inputs.Get("sources")
	if err != nil {
		return err
	}

	sources, ok := sourcesValue.(core.ArtifactID)
	if !ok {
		return errors.Errorf(
			"TypeError: wanted either filegroup or Go source target; got %T",
			sourcesValue,
		)
	}

	dependencyArtifactIDs, err := recursiveDependencies(dag)
	if err != nil {
		return errors.Wrapf(
			err,
			"Recursively collecting dependencies from %s",
			dag.ID,
		)
	}
	dependencies := make([]string, len(dependencyArtifactIDs))
	for i, dependencyArtifactID := range dependencyArtifactIDs {
		dependencies[i] = cache.Path(dependencyArtifactID)
	}

	providesValue, err := dag.Inputs.Get("provides")
	if err != nil {
		return err
	}

	providesArray, ok := providesValue.(core.FrozenArray)
	if !ok {
		return errors.Errorf(
			"TypeError: wanted list of Go compiled library targets; got %T",
			providesValue,
		)
	}

	provides := make([]string, len(providesArray))
	for i, v := range providesArray {
		if s, ok := v.(core.String); ok {
			provides[i] = string(s)
			continue
		}
		return errors.Errorf(
			"TypeError: Wanted str at index %d of 'provides' argument; "+
				"got %T",
			i,
			v,
		)
	}

	return buildLibrary(
		string(moduleName),
		provides,
		cache.Path(sources),
		dependencies,
		cache.Path(dag.ID.ArtifactID()),
		stdout,
		stderr,
	)
}

func libraryBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	link bool,
) error {
	packageName, err := dag.Inputs.GetString("package_name")
	if err != nil {
		return err
	}

	sourcesValue, err := dag.Inputs.Get("sources")
	if err != nil {
		return err
	}

	sources, ok := sourcesValue.(core.ArtifactID)
	if !ok {
		return errors.Errorf(
			"TypeError: wanted either filegroup or Go source target; got %T",
			sourcesValue,
		)
	}

	directory, err := dag.Inputs.GetString("directory")
	if err != nil {
		return err
	}

	dependenciesValue, err := dag.Inputs.Get("dependencies")
	if err != nil {
		return err
	}

	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return errors.Errorf(
			"TypeError: wanted list of Go compiled library targets; got %T",
			dependenciesValue,
		)
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating temporary directory")
	}
	defer os.Remove(tmpDir)

	targetFile := filepath.Join(tmpDir, string(packageName)+".a")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		return errors.Wrap(err, "Preparing temporary directory")
	}

	args := []string{"tool", "compile", "-pack", "-o", targetFile}

	// Add the dependency paths to the args list (-I flags)
	for i, v := range dependenciesArray {
		if dependency, ok := v.(core.ArtifactID); ok {
			args = append(args, "-I", cache.Path(dependency))
			continue
		}
		return errors.Errorf(
			"In 'dependencies' field, at index %d: TypeError: "+
				"wanted Go compiled library target; got %T",
			i,
			v,
		)
	}

	// Append all Go file paths in the sources filegroup to the arguments
	sourcesCacheDirectory := filepath.Join(
		cache.Path(sources),
		string(directory),
	)
	ctx := build.Context{
		GOARCH:   runtime.GOARCH,
		GOOS:     runtime.GOOS,
		Compiler: runtime.Compiler,
	}
	pkg, err := ctx.ImportDir(sourcesCacheDirectory, 0)
	if err != nil {
		return errors.Wrap(err, "Collecting source files")
	}
	// jsonPrettyPrint(pkg)
	for _, files := range [][]string{
		pkg.GoFiles,
		// pkg.SFiles,
	} {
		for _, file := range files {
			args = append(args, filepath.Join(sourcesCacheDirectory, file))
		}
	}

	fmt.Fprintf(stderr, "Running: go %s\n", strings.Join(args, " "))
	cmd := exec.Command("go", args...)
	cmd.Env = append(
		os.Environ(),
		"GOOS=darwin",
		"GOARCH=amd64",
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Compiling source library")
	}

	finalCachePath := cache.Path(dag.ID.ArtifactID())
	if err := os.MkdirAll(filepath.Dir(finalCachePath), 0755); err != nil {
		return errors.Wrap(err, "Making parent directory in cache")
	}

	// if link is true, build the main executable and put the executable file
	// in the cache, otherwise move the archive into the cache.
	if link {
		dependencies, err := recursiveDependencies(dag)
		if err != nil {
			return errors.Wrap(err, "Recursively fetching dependencies")
		}

		args = []string{"tool", "link"}
		for _, dependency := range dependencies {
			args = append(args, "-L", cache.Path(dependency))
		}
		args = append(args, "-o", cache.Path(dag.ID.ArtifactID()), targetFile)
		fmt.Fprintf(stderr, "Running: go %s\n", strings.Join(args, " "))
		cmd := exec.Command("go", args...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "Compiling source library")
		}
	} else {
		finalCachePath = filepath.Join(
			finalCachePath,
			"pkg",
			string(packageName)+".a",
		)
		if err := os.Rename(tmpDir, finalCachePath); err != nil {
			return errors.Wrap(err, "Moving temp dir into final cache location")
		}
	}

	return nil
}

func recursiveDependencies(dag core.DAG) ([]core.ArtifactID, error) {
	var output []core.ArtifactID

	dependenciesValue, err := dag.Inputs.Get("dependencies")
	if err != nil {
		return nil, errors.Wrapf(err, "Scanning dependencies on %s", dag.ID)
	}

	dependenciesArray, ok := dependenciesValue.(core.FrozenArray)
	if !ok {
		return nil, errors.Errorf(
			"TypeError: expected list of Go compiled library targets; "+
				"found %T",
			dependenciesValue,
		)
	}

OUTER:
	for i, dependencyValue := range dependenciesArray {
		if dependencyID, ok := dependencyValue.(core.ArtifactID); ok {
			// Now that we know the current item in the dependency array is in
			// fact an ArtifactID, let's find the corresponding DAG in
			// dag.Dependencies such that we can recursively collect *its*
			// dependencies and attach them to `output`.
			for _, dependencyDAG := range dag.Dependencies {
				if dependencyDAG.ID.ArtifactID() == dependencyID {
					transitiveDeps, err := recursiveDependencies(dependencyDAG)
					if err != nil {
						return nil, errors.Wrapf(
							err,
							"Scanning dependency %s of %s",
							dependencyDAG.ID,
							dag.ID,
						)
					}

					// We've collected the dependencies of the current
					// dependencies; let's add them to `output` and move on to
					// the next dependency in `dependenciesArray`.
					output = append(output, transitiveDeps...)
					continue OUTER
				}
			}
			continue
		}
		return nil, errors.Errorf(
			"TypeError: Index %d: Expected Go compiled targets; found %T",
			i,
			dependencyValue,
		)
	}

	return append(output, dag.ID.ArtifactID()), nil
}

var Library = core.Plugin{
	Type: "go_module",
	Factory: func(core.FrozenObject) (core.BuildScript, error) {
		return func(
			dag core.DAG,
			cache core.Cache,
			stdout io.Writer,
			stderr io.Writer,
		) error {
			return libraryBuildScript2(dag, cache, stdout, stderr)
		}, nil
	},
}

var Binary = core.Plugin{
	Type: "go_binary",
	Factory: func(core.FrozenObject) (core.BuildScript, error) {
		return func(
			dag core.DAG,
			cache core.Cache,
			stdout io.Writer,
			stderr io.Writer,
		) error {
			return libraryBuildScript(dag, cache, stdout, stderr, true)
		}, nil
	},
}
