load("lib://py", pypi = "py_pypi_library", virtualenv = "py_virtualenv")
load(
    "3rdParty/python",
    "atomicwrites",
    "attrs",
    "packaging",
    "pluggy",
    "py",
    "wcwidth",
)

# It's important that this is defined outside of pytest() so that it is a
# member of this package and not the package that invokes pytest(). The latter
# case would mean that every invoking package would have its own copy of
# pytest (which implies that every invoking package would have to *build* its
# own copy of pytest).
_pytest = pypi(
    name = "pytest",
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
    if directory == None:
        directory = ""
    if dependencies == None:
        dependencies = []
    dependencies.append(_pytest)
    return mktarget(
        name = name,
        args = {
            "sources": sources,
            "directory": directory,
            "dependencies": virtualenv(
                name = "{}_dependencies".format(name),
                dependencies = dependencies,
            ),
        },
        type = "py_test",
    )
