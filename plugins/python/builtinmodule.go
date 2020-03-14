package python

const BuiltinModule = `
load("std/command", "bash")

def pypi(name, pypi_name = None, constraint = None, dependencies = None):
    dependencies = dependencies if dependencies != None else []
    return bash(
        name = name,
        environment = {
            "DEPENDENCY_{}".format(i): dependency
            for i, dependency in enumerate(dependencies)
        },
        script = "\n".join(
            [
                "python -m pip wheel --no-deps -w $OUTPUT {}{}".format(
                    pypi_name if pypi_name != None else name,
                    constraint if constraint != None else "",
                ),
                'touch "$OUTPUT/DEPENDENCIES"',
            ] + [
                'echo "$DEPENDENCY_{}" >> "$OUTPUT/DEPENDENCIES"'.format(i)
                for i, _ in enumerate(dependencies)
            ],
        ),
    )

def venv(name, dependencies = None):
    return mktarget(
        name = name,
        type = "virtualenv",
        args = {
            "dependencies": dependencies if dependencies != None else [],
        },
    )

_pex = bash(
    name = "__pex__",
    script = """
python -m venv .venv
source .venv/bin/activate
python -m pip install pex
python -m pex --disable-cache --python python3.6 pex -o $OUTPUT -c pex
    """,
)

def pex(
    name,
    entry_point,
    bin_package,
    bin_package_name = None,
    dependencies = None,
):
    dependencies = {
        "DEPENDENCY_{}".format(i): dependency
        for i, dependency in enumerate(dependencies)
    } if dependencies != None else {}

    bin_package_name = bin_package_name if bin_package_name != None else name
    environment = dict(dependencies)
    environment["PEX"] = _pex
    environment["BIN"] = bin_package
    return bash(
        name = name,
        environment = environment,
        script = """
function fetchDeps() {{
    for dep in $@; do
        echo $dep
        fetchDeps $(cat $dep/DEPENDENCIES)
    done
}}

# Get full dependency list (including transitive deps) and put into the local
# env var $deps
deps=$(fetchDeps {} $BIN | sort | uniq)

# Get the wheel for each dep
wheels=$(for dep in $deps; do echo $dep/$(ls $dep | grep '.whl'); done)

# Build a pex file from the wheels, storing it in $OUTPUT and setting the
# package/entrypoint appropriately
$PEX --disable-cache --python python3.6 --no-index $wheels -o $OUTPUT -e {}:{}
""".format(
            " ".join(["${}".format(k) for k in dependencies.keys()]),
            bin_package_name,
            entry_point,
        ),
    )

atomicwrites = pypi(name = "atomicwrites")
attrs = pypi(name = "attrs")
pyparsing = pypi(name = "pyparsing", constraint = "==2.4.5")
packaging = pypi(
    name = "packaging",
    constraint = "==19.2",
    dependencies = [pyparsing],
)
six = pypi(name = "six")
more_itertools = pypi(name = "more-itertools", dependencies = [ six ])
zipp = pypi(name = "zipp", dependencies = [ more_itertools ])
importlib_metadata = pypi(name = "importlib_metadata", dependencies = [ zipp ])
pluggy = pypi(name = "pluggy", dependencies = [ importlib_metadata ])
py = pypi(name = "py")
wcwidth = pypi(name = "wcwidth")

_pytest_wheel = pypi(
    name = "__pytest_wheel__",
    pypi_name = "pytest",
    constraint = "==5.2.0",
	dependencies = [
		atomicwrites,
		attrs,
		packaging,
		pluggy,
		py,
		wcwidth,
	],
)

def pytest(name, sources, directory = None, dependencies = None):
    return bash(
        name = name,
        environment = {
            "SOURCES": sources,
            "PYTEST": pex(
                name = "{}_dependencies".format(name),
                bin_package = _pytest_wheel,
                bin_package_name = "pytest",
                entry_point = "main",
                dependencies = dependencies,
            ),
        },
        script = '$PYTEST $SOURCES{} | tee $OUTPUT'.format(
            "/"+directory if directory != None else "",
        ),
    )

def py_source_binary(
    name,
    sources,
    entry_point,
    package_name = None,
    dependencies = None,
):
    return pex(
        name = name,
        bin_package = py_source_library(
            name = "{}_sources".format(name),
            package_name = package_name,
            sources = sources,
            dependencies = dependencies,
        ),
        bin_package_name = package_name,
        entry_point = entry_point,
    )

def py_source_library(name, sources, package_name = None, dependencies = None):
    dependencies = dependencies if dependencies != None else []
    environment = {
        "DEPENDENCY_{}".format(i): dependency
        for i, dependency in enumerate(dependencies)
    }
    environment["SOURCES"] = sources
    return bash(
        name = name,
        environment = environment,
        script = "\n".join(
            [
                "python -m pip wheel --no-cache-dir -w $OUTPUT $SOURCES",
                'touch "$OUTPUT/DEPENDENCIES"',
            ] + [
                'echo "$DEPENDENCY_{}" >> "$OUTPUT/DEPENDENCIES"'.format(i)
                for i, _ in enumerate(dependencies)
            ],
        ),
    )
`
