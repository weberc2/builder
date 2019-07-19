def py_source_binary(
    name,
    sources,
    entry_point,
    package_name = None,
    dependencies = None,
):
    if package_name == None:
        package_name = name
    if dependencies == None:
        dependencies = []
    return mktarget(
        name = name,
        args = {
            "package_name": package_name,
            "sources": sources,
            "entry_point": entry_point,
            "dependencies": dependencies,
        },
        type = "py_source_binary",
    )

def py_source_library(name, sources, package_name = None, dependencies = None):
    if dependencies == None:
        dependencies = []
    if package_name == None:
        package_name = name
    return mktarget(
        name = name,
        args = {
            "package_name": package_name,
            "sources": sources,
            "dependencies": dependencies,
        },
        type = "py_source_library",
    )

def py_pypi_library(
    name,
    package_name = None,
    constraint = None,
    dependencies = None,
):
    if package_name == None:
        package_name = name
    if constraint == None:
        constraint = ""
    if dependencies == None:
        dependencies = []
    return mktarget(
        name = name,
        args = {
            "package_name": package_name,
            "constraint": constraint,
            "dependencies": dependencies,
        },
        type = "py_pypi_library",
    )

def py_test(name, sources, directory = None, dependencies = None):
    if directory == None:
        directory = ""
    if dependencies == None:
        dependencies = []
    dependencies.append(
        py_pypi_library(
            name = "pytest",
            dependencies = [
                py_pypi_library(name = "packaging"),
                py_pypi_library(name = "attrs"),
                py_pypi_library(name = "more-itertools"),
                py_pypi_library(name = "atomicwrites"),
                py_pypi_library(name = "pluggy"),
                py_pypi_library(name = "importlib-metadata"),
                py_pypi_library(name = "wcwidth"),
                py_pypi_library(name = "pyparsing"),
                py_pypi_library(name = "six"),
                py_pypi_library(name = "zipp"),
            ],
        ),
    )
    return mktarget(
        name = name,
        args = {
            "sources": sources,
            "directory": directory,
            "dependencies": dependencies,
        },
        type = "py_test",
    )