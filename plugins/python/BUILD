load("lib://py", pypi = "py_pypi_library", _py_test = "py_test")
load(
    "3rdParty/python",
    "atomicwrites",
    "attrs",
    "packaging",
    "pluggy",
    "py",
    "wcwidth",
)

_pytest = pypi(
    name = "pytest",
    dependencies = [ atomicwrites, attrs, packaging, pluggy, py, wcwidth ],
)

def py_test(name, sources, directory = None, dependencies = None):
    return _py_test(
        name = name,
        sources = sources,
        pytest = _pytest,
        directory = directory,
        dependencies = dependencies,
    )