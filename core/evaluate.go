package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

// Evaluator evaluates the macro language into distinct target definitions.
type Evaluator struct {
	// PackageRoot is the directory that contains all packages.
	PackageRoot string

	// LibRoot is the directory that contains all libraries.
	LibRoot string
}

type entry struct {
	globals starlark.StringDict
	err     error
}

type cache map[string]*entry

func cacheLoad(
	cache map[string]*entry,
	load func(th *starlark.Thread, mod string) (starlark.StringDict, error),
) func(th *starlark.Thread, mod string) (starlark.StringDict, error) {
	return func(th *starlark.Thread, mod string) (starlark.StringDict, error) {
		e, ok := cache[mod]
		if e == nil {
			if ok {
				// request for package whose loading is in progress
				return nil, fmt.Errorf("cycle in load graph")
			}

			// Add a placeholder to indicate "load in progress".
			cache[mod] = nil

			// Do the load
			globals, err := load(th, mod)
			e = &entry{globals, err}
			cache[mod] = e
		}

		return e.globals, e.err
	}
}

func loadLibrary(
	cache map[string]*entry,
	libroot string,
	lib string,
) (starlark.StringDict, error) {
	globals, err := starlark.ExecFile(
		&starlark.Thread{
			Name: lib,
			Load: cacheLoad(
				cache,
				func(
					th *starlark.Thread,
					lib string,
				) (starlark.StringDict, error) {
					return loadLibrary(cache, libroot, lib)
				},
			),
		},
		filepath.Join(libroot, lib[len("lib://"):]+".star"),
		nil,
		starlark.StringDict{
			"mktarget": starlark.NewBuiltin("mktarget", mktarget),
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Loading %s", lib)
	}

	return globals, nil
}

func loadPackage(
	cache map[string]*entry,
	libroot string,
	pkgroot string,
	pkg string,
) (starlark.StringDict, error) {
	return starlark.ExecFile(
		&starlark.Thread{
			Name: pkg,
			Load: cacheLoad(
				cache,
				func(
					th *starlark.Thread,
					pkg string,
				) (starlark.StringDict, error) {
					return load(cache, libroot, pkgroot, pkg)
				},
			),
		},
		filepath.Join(pkgroot, pkg, "BUILD"),
		nil,
		starlark.StringDict{
			"mktarget": starlark.NewBuiltin("mktarget", mktarget),
			"glob":     starlark.NewBuiltin("glob", glob),
		},
	)
}

func load(
	cache cache,
	libroot string,
	pkgroot string,
	mod string,
) (starlark.StringDict, error) {
	if strings.HasPrefix(mod, "lib://") {
		return loadLibrary(cache, libroot, mod)
	}

	globals, err := loadPackage(cache, libroot, pkgroot, mod)
	if err != nil {
		return nil, errors.Wrapf(err, "Loading %s", mod)
	}

	return globals, nil
}

func (ev Evaluator) Evaluate(p PackageName) ([]Target, error) {
	globals, err := loadPackage(
		map[string]*entry{},
		ev.LibRoot,
		ev.PackageRoot,
		string(p),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Loading %s", p)
	}

	var targets []Target
	for _, global := range globals {
		if t, ok := global.(Target); ok {
			targets = append(targets, t)
		}
	}

	return targets, nil
}

func findKwarg(kwargs []starlark.Tuple, kw string) (starlark.Value, error) {
	for _, kwarg := range kwargs {
		if kwarg[0] == starlark.String(kw) {
			return kwarg[1], nil
		}
	}
	return nil, fmt.Errorf("Missing argument: '%s'", kw)
}

func mktarget(
	th *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("target() only takes keyword args")
	}

	if strings.HasPrefix(th.Name, "lib://") {
		return nil, fmt.Errorf("mktarget() invoked by library '%s'", th.Name)
	}

	var t Target
	t.ID.Package = PackageName(th.Name)

	nameValue, err := findKwarg(kwargs, "name")
	if err != nil {
		return nil, err
	}
	if name, ok := nameValue.(starlark.String); ok {
		if strings.Contains(string(name), "/") {
			return nil, fmt.Errorf("ValueError: Invalid value for 'name'")
		}
		t.ID.Target = TargetName(name)
	} else {
		return nil, fmt.Errorf(
			"TypeError: 'name' must be str, got %T",
			nameValue,
		)
	}

	typeValue, err := findKwarg(kwargs, "type")
	if err != nil {
		return nil, err
	}
	if typ, ok := typeValue.(starlark.String); ok {
		t.BuilderType = BuilderType(typ)
	} else {
		return nil, fmt.Errorf(
			"TypeError: 'type' must be str, got %T",
			typeValue,
		)
	}

	argsValue, err := findKwarg(kwargs, "args")
	if err != nil {
		return nil, err
	}
	if args, ok := argsValue.(*starlark.Dict); ok {
		inputs, err := starlarkDictToObject(t.ID, args)
		if err != nil {
			return nil, err
		}
		t.Inputs = inputs
	} else {
		return nil, fmt.Errorf(
			"TypeError: 'args' must be a dict, got %T",
			argsValue,
		)
	}

	return t, nil
}

func starlarkValueToInput(tid TargetID, value starlark.Value) (Input, error) {
	switch x := value.(type) {
	case FileGroup:
		x.Package = tid.Package
		return x, nil
	case Target:
		return x, nil
	case Input:
		return x, nil
	case starlark.String:
		return String(x), nil
	case starlark.Int:
		i, ok := x.Int64()
		if !ok {
			panic("Error converting starlark.Int to int64!")
		}
		return Int(i), nil
	case starlark.Bool:
		return Bool(x), nil
	case *starlark.Dict:
		return starlarkDictToObject(tid, x)
	case *starlark.List:
		return starlarkListToArray(tid, x)
	default:
		return nil, fmt.Errorf("TypeError: Invalid argument type %T", x)
	}
}

func starlarkListToArray(tid TargetID, l *starlark.List) (Array, error) {
	a := make(Array, l.Len())
	for i := 0; i < l.Len(); i++ {
		input, err := starlarkValueToInput(tid, l.Index(i))
		if err != nil {
			return nil, err
		}
		a[i] = input
	}
	return a, nil
}

func starlarkDictToObject(tid TargetID, d *starlark.Dict) (Object, error) {
	keys := d.Keys()
	out := make(Object, len(keys))

	for i, keyValue := range keys {
		if key, ok := keyValue.(starlark.String); ok {
			value, found, err := d.Get(key)
			if err != nil {
				return nil, err
			}
			if !found {
				panic(fmt.Sprintf(
					"starlark.Dict reports key %s but value not found",
					key,
				))
			}

			input, err := starlarkValueToInput(tid, value)
			if err != nil {
				return nil, err
			}

			out[i] = Field{Key: string(key), Value: input}
		}
	}

	return out, nil
}

func glob(
	th *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	if len(kwargs) > 0 {
		return nil, fmt.Errorf("Unexpected keyword argument")
	}

	patterns := make([]string, len(args))
	for i, arg := range args {
		if s, ok := arg.(starlark.String); ok {
			patterns[i] = string(s)
		}
	}

	return FileGroup{Package: PackageName(th.Name), Patterns: patterns}, nil
}
