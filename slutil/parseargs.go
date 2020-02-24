package slutil

import (
	"fmt"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

type TypeErr struct {
	Wanted string
	Got    string
}

func (err TypeErr) Error() string {
	return fmt.Sprintf("TypeError: expected %s, found %s", err.Wanted, err.Got)
}

func NewTypeErr(wanted string, v interface{}) TypeErr {
	return TypeErr{Wanted: wanted, Got: fmt.Sprintf("%T", v)}
}

type PosSpec struct {
	Keyword starlark.String
	Value   func(starlark.Value) error
}

type KwSpec struct {
	Keyword starlark.String
	Value   func(starlark.Value) error
	Default starlark.Value
}

type ArgsSpec struct {
	PosSpecs []PosSpec
	KwSpecs  []KwSpec
}

type Args struct {
	Pos starlark.Tuple
	Kw  []starlark.Tuple
}

type WrongPosArgCountErr struct {
	Fn             string
	TakesPosArgs   int
	TakesTotalArgs int
	GivenPosArgs   int
}

func (err WrongPosArgCountErr) Error() string {
	if err.TakesPosArgs == err.TakesTotalArgs {
		return fmt.Sprintf(
			"%s() takes %d positional argument(s), but %d were given",
			err.Fn,
			err.TakesPosArgs,
			err.GivenPosArgs,
		)
	}
	return fmt.Sprintf(
		"%s() takes from %d to %d positional argument(s), but %d were given",
		err.Fn,
		err.TakesPosArgs,
		err.TakesTotalArgs,
		err.GivenPosArgs,
	)
}

type MissingRequiredArgumentsErr struct {
	Fn          string
	MissingHead string
	MissingTail []string
}

func (err MissingRequiredArgumentsErr) Error() string {
	list := fmt.Sprintf("'%s'", err.MissingHead)
	switch len(err.MissingTail) {
	case 0:
	case 1:
		list = fmt.Sprintf(
			"%s and '%s'",
			list,
			err.MissingTail[0],
		)
	default:
		for _, arg := range err.MissingTail[:len(err.MissingTail)-1] {
			list = fmt.Sprintf("%s, '%s'", list, arg)
		}
		list = fmt.Sprintf(
			"%s, and '%s'",
			list,
			err.MissingTail[len(err.MissingTail)-1],
		)
	}

	return fmt.Sprintf(
		"%s() missing %d required positional arguments: %s",
		err.Fn,
		len(err.MissingTail)+1,
		list,
	)
}

type MultipleValuesForArgErr struct {
	Fn  string
	Arg string
}

func (err MultipleValuesForArgErr) Error() string {
	return fmt.Sprintf(
		"%s() got multiple values for argument '%s'",
		err.Fn,
		err.Arg,
	)
}

type UnexpectedKeywordArgErr struct {
	Fn  string
	Arg string
}

func (err UnexpectedKeywordArgErr) Error() string {
	return fmt.Sprintf(
		"%s() got an unexpected keyword argument '%s'",
		err.Fn,
		err.Arg,
	)
}

func ParseArgs(fn string, args Args, spec ArgsSpec) error {
	values := make([]starlark.Value, len(spec.PosSpecs)+len(spec.KwSpecs))
	if len(args.Pos) > len(values) {
		return WrongPosArgCountErr{
			Fn:             fn,
			TakesPosArgs:   len(spec.PosSpecs),
			TakesTotalArgs: len(values),
			GivenPosArgs:   len(args.Pos),
		}
	}

	// Initialize keyword arguments with default values. We will overwrite them
	// as necessary in the next loops.
	for i, kwSpec := range spec.KwSpecs {
		values[len(spec.PosSpecs)+i] = kwSpec.Default
	}

	// Set values from positional args. If some kwargs are set positionally,
	// this will overwrite their default values.
	for i, pos := range args.Pos {
		values[i] = pos
	}

	// Iterate through the provided keyword arguments. We'll check to see if
	// they set a kwarg or an arg (error if neither and error if it has already
	// been set).
NEXT_KW:
	for i, kw := range args.Kw {
		// Check for duplicate kwargs.
		for j, kw2 := range args.Kw {
			if kw[0] == kw2[0] && i != j {
				return MultipleValuesForArgErr{
					Fn:  fn,
					Arg: string(kw[0].(starlark.String)),
				}
			}
		}

		for i, posSpec := range spec.PosSpecs {
			if posSpec.Keyword == kw[0].(starlark.String) {
				if i < len(args.Pos) {
					return MultipleValuesForArgErr{
						Fn:  fn,
						Arg: string(posSpec.Keyword),
					}
				}
				values[i] = kw[1]
				continue NEXT_KW
			}
		}

		for i, kwSpec := range spec.KwSpecs {
			if kwSpec.Keyword == kw[0].(starlark.String) {
				if i < len(args.Pos) {
					return MultipleValuesForArgErr{
						Fn:  fn,
						Arg: string(kwSpec.Keyword),
					}
				}
				values[len(spec.PosSpecs)+i] = kw[1]
				continue NEXT_KW
			}
		}

		return UnexpectedKeywordArgErr{
			Fn:  fn,
			Arg: string(kw[0].(starlark.String)),
		}
	}

	// In case there weren't enough positionally-provided arguments to cover
	// the positional args, then we are relying on the keyword-provided
	// arguments to fill in the gap (this only applies to positional arguments
	// because they don't have default values by definition).
	if len(args.Pos) < len(spec.PosSpecs) {
		var missing []string

		// We can skip the first len(args.Pos) values because these are
		// guaranteed to be filled by the positionally-provided arguments.
		// These are necessarily non-nil (except for programmer error).
		for i, value := range values[len(args.Pos):len(spec.PosSpecs)] {
			if value == nil {
				missing = append(
					missing,
					string(spec.PosSpecs[len(args.Pos)+i].Keyword),
				)
			}
		}

		if len(missing) > 0 {
			return MissingRequiredArgumentsErr{
				Fn:          fn,
				MissingHead: missing[0],
				MissingTail: missing[1:],
			}
		}
	}

	for i, posSpec := range spec.PosSpecs {
		if err := posSpec.Value(values[i]); err != nil {
			return errors.Wrapf(err, "Parsing argument %s", posSpec.Keyword)
		}
	}
	for i, kwSpec := range spec.KwSpecs {
		if err := kwSpec.Value(values[len(spec.PosSpecs)+i]); err != nil {
			return errors.Wrapf(err, "Parsing argument %s", kwSpec.Keyword)
		}
	}
	return nil
}

func AssertString(f func(s string) error) func(starlark.Value) error {
	return func(v starlark.Value) error {
		if s, ok := v.(starlark.String); ok {
			return f(string(s))
		}
		return NewTypeErr("str", v)
	}
}

func ParseString(sptr *string) func(starlark.Value) error {
	return AssertString(func(s string) error {
		*sptr = s
		return nil
	})
}

func AssertDict(f func(d *starlark.Dict) error) func(starlark.Value) error {
	return func(v starlark.Value) error {
		if d, ok := v.(*starlark.Dict); ok {
			return f(d)
		}
		return NewTypeErr("dict", v)
	}
}

func AssertDictOf(
	f func(starlark.Value, starlark.Value) error,
) func(starlark.Value) error {
	return AssertDict(func(d *starlark.Dict) error {
		for _, key := range d.Keys() {
			val, found, err := d.Get(key)
			if !found {
				panic(fmt.Sprintf(
					"Couldn't find key '%s', but key came from Keys()",
					key,
				))
			}
			if err != nil {
				panic(fmt.Sprintf(
					"Error retrieving key '%s' from dict: %v",
					key,
					err,
				))
			}
			if err := f(key, val); err != nil {
				return errors.Wrapf(err, "Processing key %s", key)
			}
		}
		return nil
	})
}

func AssertObjectOf(
	f func(string, starlark.Value) error,
) func(starlark.Value) error {
	return AssertDictOf(func(k, v starlark.Value) error {
		if s, ok := k.(starlark.String); ok {
			return f(string(s), v)
		}
		return NewTypeErr("str", k)
	})
}
