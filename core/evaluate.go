package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	sl "github.com/weberc2/builder/slutil"
	"go.starlark.net/starlark"
)

// Evaluator evaluates the macro language into distinct target definitions.
type Evaluator struct {
	// PackageRoot is the directory that contains all packages.
	PackageRoot string

	// BuiltinModules is a list of modules that are baked into the application
	// process.
	BuiltinModules map[string]string
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

var ErrUnknownBuiltinModule = errors.Errorf("Unknown builtin module")

type UnknownBuiltinModuleErr string

func (err UnknownBuiltinModuleErr) Error() string {
	return fmt.Sprintf("Unknown builtin module: %s", string(err))
}

func loadBuiltin(
	cache map[string]*entry,
	builtinModules map[string]string,
	builtin string,
) (starlark.StringDict, error) {
	if script, found := builtinModules[builtin]; found {
		return starlark.ExecFile(
			&starlark.Thread{
				Name: builtin,
				Load: cacheLoad(
					cache,
					func(
						th *starlark.Thread,
						lib string,
					) (starlark.StringDict, error) {
						return loadBuiltin(cache, builtinModules, lib)
					},
				),
			},
			"builtin://"+builtin,
			script,
			starlark.StringDict{
				"mktarget": starlark.NewBuiltin("mktarget", mktarget),
			},
		)
	}
	return nil, UnknownBuiltinModuleErr(builtin)
}

func loadPackage(
	cache map[string]*entry,
	builtinModules map[string]string,
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
					return load(cache, builtinModules, pkgroot, pkg)
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
	builtinModules map[string]string,
	pkgroot string,
	mod string,
) (starlark.StringDict, error) {
	globals, err := loadBuiltin(cache, builtinModules, mod)
	if _, ok := err.(UnknownBuiltinModuleErr); ok {
		globals, err = loadPackage(cache, builtinModules, pkgroot, mod)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "Loading %s", mod)
	}

	return globals, nil
}

func (ev Evaluator) Evaluate(p PackageName) ([]Target, error) {
	globals, err := loadPackage(
		map[string]*entry{},
		ev.BuiltinModules,
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
	t := Target{ID: TargetID{Package: PackageName(th.Name)}}
	return t, sl.ParseArgs(
		"mktarget",
		sl.Args{Pos: args, Kw: kwargs},
		sl.ArgsSpec{
			PosSpecs: []sl.PosSpec{{
				Keyword: "name",
				Value: sl.AssertString(func(s string) error {
					if strings.Contains(s, "/") {
						return errors.New(
							"ValueError: Invalid value for 'name'",
						)
					}
					t.ID.Target = TargetName(s)
					return nil
				}),
			}, {
				Keyword: "type",
				Value: sl.AssertString(func(s string) error {
					t.BuilderType = BuilderType(s)
					return nil
				}),
			}, {
				Keyword: "args",
				Value: sl.AssertDict(func(d *starlark.Dict) error {
					inputs, err := starlarkDictToObject(t.ID, d)
					if err != nil {
						return errors.Wrap(err, "Parsing target args")
					}
					t.Inputs = inputs
					return nil
				}),
			}},
		},
	)
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
