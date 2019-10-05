# `builder`

## Concepts

### Repository / workspace

`builder` doesn't know about version control systems (e.g., git) directly, but
from a software project management perspective, the root of the git repository
is typically also the root of a `builder` project. `builder` refers to this
directory and all its contents as the *repository* or *workspace*, and the
directory itself is referred to as the *repository root* or *workspace root*.
All `builder` commands are relative to the repository root. `builder`
identifies the repository by looking at the current working directory and
recursing upward until it finds a directory that contains a `WORKSPACE` file.

In the `builder` addressing scheme, the repository root's address is `//`.

### Targets

The central concept in `builder` is the *target*. Targets are the abstract
representation of an artifact to be built. They have inputs and a type which
tells `builder` what to do to those inputs to build the concrete artifact. The
type can be thought of as a *build function* that takes inputs and outputs an
artifact. The specific types of inputs and the type of the output artifact vary
from target type to target type. For example, a `python_library` target type
takes inputs such as Python source files and other `python_library` target
dependencies and outputs a Python wheel artifact.

Specific input types include Strings, Ints, [other targets](#dependencies), and
[file groups](#file-groups).

### Packages

Targets are scoped to a *package*--a directory containing a `BUILD` file which
defines the targets in the macro language, Starlark. Targets may only reference
files inside of their package directory.

Packages are addressed from the repository root: `//path/to/package` and
targets inside of the package are addressed via: `//path/to/package:target1`.

### File groups

*File groups* are collections of source files inside of a given package which
are depended upon by targets inside of the package. These are the leaf nodes in
the dependency graph. They are defined by the `glob()` function in the macro
language.

For example, a `python_library` target may be defined as:

```starlark
python_library(
    # ...
    sources = glob("setup.py", "src/**/*.py"),
)
```

This implies that the target depends on the `setup.py` file and any `.py` file
under the `src` directory (these paths are relative to the package root). If
any of these files change, this target (and any targets that depend on it) will
be rebuilt.

### Dependencies

Targets depend on file groups inside of their own package and targets. Target
dependencies can come from the target's own package or they can be imported
from other packages. Any target that is referenced anywhere in another target's
set of inputs is implicitly a dependency of the target that references it.

Here is an example of a package-local dependency:

```starlark
foo = python_library(
    # ...
)

# Target `bar` depends on `foo`
bar = python_library(
    # ...
    dependencies = [ foo ],
)
```

Here is an example of a dependency from another package:

```starlark
load("3rdParty/python", "requests")

# Target `baz` depends on `//3rdParty/python:requests` (the `requests` target
# from the `//3rdParty/python` package).
baz = python_library(
    # ...
    dependencies = [ requests ],
)
```

### Hashing

By walking a target's inputs for dependencies, `builder` can assemble a
dependency graph for a given target. This dependency graph is the key to fast
and correct builds--because `builder` requires that all resources used for a
build be declared as inputs, it can safely use cached build artifacts so long
as none of the target's dependencies have changed. Put differently, the
dependency graph allows `builder` to determine whether or not its cache
contains an artifact for the target at a given set of inputs.

This caching mechanism is straightforward--`builder` stores artifacts in
the cache using the (recursive) hash of the target's inputs as the key. String
and Int inputs are hashed straightforwardly. The hash of a file group input is
the hash of each file's contents and metadata. The hash of a dependency target
input is computed by the same process (hence recursive). This means that any
change to any input (whether String or Int or file group or dependency target)
will result in a different hash. If `builder` can't find an artifact for that
hash in the build cache, it will rebuild that artifact.

### Targets vs frozen targets

TODO: Is the user documentation the right place for this?

A *frozen target* is a snapshot of a target and everything it references at a
given point in time. The process for freezing a target is as follows:

1. For primitive inputs (like String and Int), do nothing (the corresponding
   inputs on the frozen structure are exactly the same as the unfrozen
   structure).
1. For file group inputs, read and hash the data and metadata for every file in
   the group, copy the files into the build cache (storing them at their hash),
   and replace the file group inputs with cache references that point to the
   file group's location in the build cache.
1. For dependency target inputs, recursively freeze the dependency targets and
   replace the dependency targets with their frozen equivalents.
1. Lastly, recursively hash all of the inputs and hash the hashes. Put the
   resulting hash onto the frozen target structure. This hash uniquely
   identifies the frozen target (it uniquely identifies the target at the
   point in time at which it was frozen).

The frozen target snapshot can then be used to rebuild the exact same artifact
over and over again (even if the source files change in the repository), and
the hash for the frozen target is used to seed the location of the resulting
artifact in the build cache (and therefore can be used to determine if a
rebuild is necessary).

## Macro language

TODO

### Build Pipeline

TODO (Eval -> Freeze -> Execute)