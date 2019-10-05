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

def py_test(name, sources, pytest, directory = None, dependencies = None):
    if directory == None:
        directory = ""
    if dependencies == None:
        dependencies = []
    dependencies.append(pytest)
    return mktarget(
        name = name,
        args = {
            "sources": sources,
            "directory": directory,
            "dependencies": py_virtualenv(
                name = "{}_dependencies".format(name),
                dependencies = dependencies,
            ),
        },
        type = "py_test",
    )

def py_virtualenv(name, dependencies):
    return mktarget(
        name = name,
        args = {"dependencies": dependencies},
        type = "py_virtualenv",
    )