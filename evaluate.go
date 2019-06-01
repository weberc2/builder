package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"go.starlark.net/starlark"
)

type Evaluator struct{ Root string }

func (ev Evaluator) Evaluate(p PackageName) ([]Target, error) {
	var targets targets
	builtins := starlark.StringDict{
		"file": starlark.NewBuiltin(
			"file",
			func(
				th *starlark.Thread,
				_ *starlark.Builtin,
				args starlark.Tuple,
				kwargs []starlark.Tuple,
			) (starlark.Value, error) {
				if len(kwargs) > 0 {
					return nil, fmt.Errorf("Unexpected keyword argument")
				}
				if len(args) != 1 {
					return nil, fmt.Errorf(
						"Expected 1 unnamed argument; found %d",
						len(args),
					)
				}

				if s, ok := args[0].(starlark.String); ok {
					return SourcePath{FilePath: string(s)}, nil
				}

				return nil, fmt.Errorf(
					"TypeError: Expected str, got %T",
					args[0],
				)
			},
		),
		"mktarget": starlark.NewBuiltin("mktarget", targets.newTarget),
		"reftarget": starlark.NewBuiltin(
			"reftarget",
			func(
				_ *starlark.Thread,
				_ *starlark.Builtin,
				args starlark.Tuple,
				kwargs []starlark.Tuple,
			) (starlark.Value, error) {
				if len(kwargs) > 0 {
					return nil, fmt.Errorf("Unexpected keyword argument")
				}
				if len(args) != 1 {
					return nil, fmt.Errorf(
						"Expected 1 unnamed argument; found %d",
						len(args),
					)
				}

				if s, ok := args[0].(starlark.String); ok {
					return parseTargetID(string(s))
				}

				return nil, fmt.Errorf(
					"TypeError: Expected str, got %T",
					args[0],
				)
			},
		),
	}

	type entry struct {
		globals starlark.StringDict
		err     error
	}

	cache := make(map[string]*entry)
	var load func(*starlark.Thread, string) (starlark.StringDict, error)
	load = func(parent *starlark.Thread, mod string) (starlark.StringDict, error) {
		e, ok := cache[mod]
		if e == nil {
			if ok {
				// request for package whose loading is in progress
				return nil, fmt.Errorf("cycle in load graph")
			}

			// Add a placeholder to indicate "load in progress".
			cache[mod] = nil

			// Load and initialize the module in a new thread.
			filePath := filepath.Join(ev.Root, mod+".star")
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			globals, err := starlark.ExecFile(parent, filePath, data, builtins)
			e = &entry{globals, err}

			// Update the cache.
			cache[mod] = e
		}
		return e.globals, e.err
	}

	_, err := starlark.ExecFile(
		&starlark.Thread{Name: string(p), Load: load},
		filepath.Join(ev.Root, string(p), "BUILD"),
		nil,
		builtins,
	)
	return targets.targets, err
}

func findKwarg(kwargs []starlark.Tuple, kw string) (starlark.Value, error) {
	for _, kwarg := range kwargs {
		if kwarg[0] == starlark.String(kw) {
			return kwarg[1], nil
		}
	}
	return nil, fmt.Errorf("Missing argument: '%s'", kw)
}

type targets struct {
	targets []Target
}

func (ts *targets) newTarget(
	th *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("target() only takes keyword args")
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

	ts.targets = append(ts.targets, t)
	return starlark.None, nil
}

func starlarkValueToInput(tid TargetID, value starlark.Value) (Input, error) {
	switch x := value.(type) {
	case SourcePath:
		x.Package = tid.Package
		x.Target = tid.Target
		return x, nil
	case TargetID:
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
