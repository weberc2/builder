package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/adler32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.starlark.net/starlark"
)

type TargetID struct {
	Package PackageName
	Target  TargetName
}

func (tid TargetID) Freeze() {}

func (tid TargetID) String() string {
	return fmt.Sprintf("%s/%s", tid.Package, tid.Target)
}

func (tid TargetID) Truth() starlark.Bool { return starlark.Bool(true) }

func (tid TargetID) Hash() (uint32, error) {
	return JoinChecksums(
		ChecksumString(string(tid.Package)),
		ChecksumString(string(tid.Target)),
	), nil
}

func (tid TargetID) Type() string { return "Target" }

type SourcePath struct {
	Package  PackageName
	Target   TargetName
	FilePath string
}

func (sp SourcePath) ArtifactID(checksum uint32) ArtifactID {
	return ArtifactID{
		FrozenTargetID: FrozenTargetID{
			Package:  sp.Package,
			Target:   sp.Target,
			Checksum: checksum,
		},
		FilePath: sp.FilePath,
	}
}

func (sp SourcePath) Freeze() {}

func (sp SourcePath) String() string {
	return filepath.Join(string(sp.Package), string(sp.Target), sp.FilePath)
}

func (sp SourcePath) Type() string { return "SourcePath" }

func (sp SourcePath) Truth() starlark.Bool { return starlark.Bool(true) }

func (sp SourcePath) Hash() (uint32, error) {
	return JoinChecksums(
		ChecksumString(string(sp.Package)),
		ChecksumString(string(sp.Target)),
		ChecksumString(sp.FilePath),
	), nil
}

type TargetName string

type PackageName string

type Int int64

type String string

type Bool bool

type Field struct {
	Key   string
	Value Input
}

type Object []Field

type Array []Input

type Input interface {
	visitInput(InputVisitor)
}

type InputVisitor interface {
	VisitTargetID(TargetID)
	VisitSourcePath(SourcePath)
	VisitInt(int64)
	VisitString(string)
	VisitBool(bool)
	VisitObject(Object)
	VisitArray(Array)
}

func (tid TargetID) visitInput(visitor InputVisitor) {
	visitor.VisitTargetID(tid)
}

func (sp SourcePath) visitInput(visitor InputVisitor) {
	visitor.VisitSourcePath(sp)
}

func (i Int) visitInput(visitor InputVisitor) {
	visitor.VisitInt(int64(i))
}

func (s String) visitInput(visitor InputVisitor) {
	visitor.VisitString(string(s))
}

func (b Bool) visitInput(visitor InputVisitor) {
	visitor.VisitBool(bool(b))
}

func (o Object) visitInput(visitor InputVisitor) {
	visitor.VisitObject(o)
}

func (a Array) visitInput(visitor InputVisitor) {
	visitor.VisitArray(a)
}

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
					i := strings.Index(string(s), ":")
					if i < 1 { // must have a colon; can't be first character
						return nil, fmt.Errorf("ValueError: Invalid target ref")
					}

					return TargetID{
						Package: PackageName(s[:i]),
						Target:  TargetName(s[i+1:]),
					}, nil
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

type freezer struct {
	root      string
	evaluator Evaluator
	cache     Cache
}

func (f *freezer) freezeArray(a Array) ([]DAG, FrozenArray, error) {
	var deps []DAG
	out := make(FrozenArray, len(a))
	for i, elt := range a {
		dependencies, frozenElt, err := f.freezeInput(elt)
		if err != nil {
			return nil, nil, err
		}
		out[i] = frozenElt
		deps = append(deps, dependencies...)
	}
	return deps, out, nil
}

func (f *freezer) freezeSourcePath(sp SourcePath) (ArtifactID, error) {
	data, err := ioutil.ReadFile(filepath.Join(
		string(sp.Package),
		sp.FilePath,
	))
	if err != nil {
		return ArtifactID{}, err
	}

	aid := ArtifactID{
		FrozenTargetID: FrozenTargetID{
			Package: sp.Package,
			Target:  sp.Target,
			Checksum: JoinChecksums(
				ChecksumString(string(sp.Package)),
				ChecksumString(string(sp.Target)),
				ChecksumString(sp.FilePath),
				ChecksumBytes(data),
			),
		},
		FilePath: sp.FilePath,
	}

	return aid, f.cache.Write(aid, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

var ErrTargetNotFound = errors.New("Target not found")

func (f *freezer) freezeTargetID(tid TargetID) (DAG, error) {
	targets, err := f.evaluator.Evaluate(tid.Package)
	if err != nil {
		return DAG{}, err
	}

	for _, target := range targets {
		if target.ID == tid {
			dag, err := f.freezeTarget(target)
			if err != nil {
				return DAG{}, err
			}
			return dag, nil
		}
	}

	return DAG{}, ErrTargetNotFound
}

func (f *freezer) freezeInput(i Input) ([]DAG, FrozenInput, error) {
	switch x := i.(type) {
	case TargetID:
		dag, err := f.freezeTargetID(x)
		if err != nil {
			return nil, nil, err
		}
		return []DAG{dag}, dag.ID.ArtifactID(), nil
	case SourcePath:
		artifactID, err := f.freezeSourcePath(x)
		return nil, artifactID, err
	case Int:
		return nil, x, nil
	case String:
		return nil, x, nil
	case Bool:
		return nil, x, nil
	case Object:
		return f.freezeObject(x)
	case Array:
		return f.freezeArray(x)
	case nil:
		return nil, nil, nil
	}
	panic(fmt.Sprintf("Invalid input type: %T", i))
}

func (f *freezer) freezeObject(o Object) ([]DAG, FrozenObject, error) {
	var deps []DAG
	out := make(FrozenObject, len(o))
	for i, field := range o {
		dependencies, frozenValue, err := f.freezeInput(field.Value)
		if err != nil {
			return nil, nil, err
		}

		out[i] = FrozenField{Key: field.Key, Value: frozenValue}
		deps = append(deps, dependencies...)
	}
	return deps, out, nil
}

func (f *freezer) freezeTarget(t Target) (DAG, error) {
	deps, frozenInputs, err := f.freezeObject(t.Inputs)
	if err != nil {
		return DAG{}, err
	}

	return DAG{
		FrozenTarget: FrozenTarget{
			ID: FrozenTargetID{
				Package: t.ID.Package,
				Target:  t.ID.Target,
				Checksum: JoinChecksums(
					ChecksumString(string(t.ID.Package)),
					ChecksumString(string(t.ID.Target)),
					ChecksumString(string(t.BuilderType)),
					frozenInputs.checksum(),
					// TODO: Checksum the builder args
				),
			},
			Inputs:      frozenInputs,
			BuilderType: t.BuilderType,
			BuilderArgs: nil, // TODO: Freeze builder args?
		},
		Dependencies: deps,
	}, nil
}

type Target struct {
	ID          TargetID
	Inputs      Object
	BuilderType BuilderType
	BuilderArgs Object
}

type BuilderType string

type BuildScript func(inputs FrozenObject, w io.Writer) error

type Plugin struct {
	Type    BuilderType
	Factory func(args FrozenObject) (BuildScript, error)
}

type FrozenTargetID struct {
	Package  PackageName
	Target   TargetName
	Checksum uint32
}

func (ftid FrozenTargetID) String() string {
	return fmt.Sprintf("%s:%s@%d", ftid.Package, ftid.Target, ftid.Checksum)
}

func (ftid FrozenTargetID) ArtifactID() ArtifactID {
	return ArtifactID{FrozenTargetID: ftid}
}

type FrozenField struct {
	Key   string
	Value FrozenInput
}

type FrozenObject []FrozenField

type FrozenArray []FrozenInput

type ArtifactID struct {
	FrozenTargetID
	FilePath string
}

func (aid ArtifactID) String() string {
	return filepath.Join(
		string(aid.Package),
		string(aid.Target),
		string(aid.FilePath),
		fmt.Sprint(aid.Checksum),
	)
}

func (aid ArtifactID) checksum() uint32 { return aid.Checksum }

func (i Int) checksum() uint32 {
	var buf [8]byte
	binary.PutVarint(buf[:len(buf)], int64(i))
	return adler32.Checksum(buf[:len(buf)])
}

func (s String) checksum() uint32 { return ChecksumString(string(s)) }

func (b Bool) checksum() uint32 {
	if bool(b) {
		return ChecksumBytes([]byte{0})
	}
	return ChecksumBytes([]byte{1})
}

func (fo FrozenObject) checksum() uint32 {
	checksums := make([]uint32, len(fo)*2)
	for i, field := range fo {
		checksums[i*2] = ChecksumString(field.Key)
		checksums[i*2+1] = field.Value.checksum()
	}
	return JoinChecksums(checksums...)
}

func (fa FrozenArray) checksum() uint32 {
	checksums := make([]uint32, len(fa))
	for i, elt := range fa {
		checksums[i] = elt.checksum()
	}
	return JoinChecksums(checksums...)
}

type FrozenInput interface {
	frozenInput()
	checksum() uint32
}

func (aid ArtifactID) frozenInput()  {}
func (i Int) frozenInput()           {}
func (s String) frozenInput()        {}
func (b Bool) frozenInput()          {}
func (fo FrozenObject) frozenInput() {}
func (fa FrozenArray) frozenInput()  {}

type FrozenTarget struct {
	ID          FrozenTargetID
	Inputs      FrozenObject
	BuilderType BuilderType
	BuilderArgs FrozenObject
}

type ExecuteFunc func(FrozenTarget) error

func LocalExecutor(plugins []Plugin, cache Cache) ExecuteFunc {
	return func(ft FrozenTarget) error {
		for _, plugin := range plugins {
			if plugin.Type == ft.BuilderType {
				if err := cache.Exists(
					ft.ID.ArtifactID(),
				); err != ErrArtifactNotFound {
					return err
				}
				log.Printf("Missing artifact %s", ft.ID.ArtifactID())

				buildScript, err := plugin.Factory(ft.BuilderArgs)
				if err != nil {
					return err
				}

				return cache.Write(
					ft.ID.ArtifactID(),
					func(w io.Writer) error {
						log.Printf("INFO Building %s", ft.ID)
						return buildScript(ft.Inputs, w)
					},
				)
			}
		}

		return ErrPluginNotFound
	}
}

type Cache interface {
	Write(id ArtifactID, f func(io.Writer) error) error
	Exists(id ArtifactID) error
}

var ErrPluginNotFound = errors.New("Plugin not found")

type DAG struct {
	FrozenTarget
	Dependencies []DAG
}

func visitArtifactID(input FrozenInput, f func(ArtifactID) error) error {
	switch x := input.(type) {
	case FrozenObject:
		for _, field := range x {
			if err := visitArtifactID(field.Value, f); err != nil {
				return err
			}
		}
	case FrozenArray:
		for _, elt := range x {
			if err := visitArtifactID(elt, f); err != nil {
				return err
			}
		}
	case ArtifactID:
		return f(x)
	}
	return nil
}

func Build(execute ExecuteFunc, dag DAG) error {
	for _, dependency := range dag.Dependencies {
		if err := Build(execute, dependency); err != nil {
			return err
		}
	}

	return execute(dag.FrozenTarget)
}

func ChecksumBytes(bs []byte) uint32 { return adler32.Checksum(bs) }

func ChecksumString(s string) uint32 { return ChecksumBytes([]byte(s)) }

func JoinChecksums(checksums ...uint32) uint32 {
	buf := make([]byte, len(checksums)*4)
	for i, checksum := range checksums {
		binary.BigEndian.PutUint32(buf[i*4:i*4+4], checksum)
	}
	return ChecksumBytes(buf)
}

type LocalCache struct {
	Directory string
}

func (lc LocalCache) Write(id ArtifactID, f func(w io.Writer) error) error {
	artifactPath := filepath.Join(
		lc.Directory,
		string(id.Package),
		string(id.Target),
		id.FilePath,
		fmt.Sprint(id.Checksum),
	)

	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(artifactPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return f(file)
}

var ErrArtifactNotFound = errors.New("Artifact not found")

func (lc LocalCache) Exists(id ArtifactID) error {
	_, err := os.Stat(filepath.Join(
		lc.Directory,
		string(id.Package),
		string(id.Target),
		id.FilePath,
		fmt.Sprint(id.Checksum),
	))
	if os.IsNotExist(err) {
		return ErrArtifactNotFound
	}
	return err
}

func main() {
	cache := LocalCache{"/tmp/cache"}

	f := freezer{
		root:      ".",
		cache:     cache,
		evaluator: Evaluator{"."},
	}

	dag, err := f.freezeTargetID(TargetID{Package: "foo", Target: "bar"})
	if err != nil {
		log.Fatal(err)
	}

	if err := Build(
		LocalExecutor(
			[]Plugin{{
				Type: "noop",
				Factory: func(args FrozenObject) (BuildScript, error) {
					return func(inputs FrozenObject, w io.Writer) error {
						_, err := fmt.Fprintln(w, "Done!")
						return err
					}, nil
				},
			}},
			cache,
		),
		dag,
	); err != nil {
		log.Fatal(err)
	}
}
