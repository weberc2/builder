# This probably won't work if there are dependencies on local packages since
# we're using modules; will need to put local dependencies in a GOPATH somehow.

def go_library(
    name,
    package_name,
    sources,
    directory = None,
    dependencies = None,
):
    if directory == None:
        directory = "/" # noop
    if dependencies == None:
        dependencies = []
    return mktarget(
        name = name,
        args = {
            "package_name": package_name,
            "sources": sources,
            "directory": directory,
            "dependencies": dependencies,
        },
        type = "go_library",
    )

def go_binary(
    name,
    package_name,
    sources,
    directory = None,
    dependencies = None,
):
    if directory == None:
        directory = "/" # noop
    if dependencies == None:
        dependencies = []
    return mktarget(
        name = name,
        args = {
            "package_name": package_name,
            "sources": sources,
            "directory": directory,
            "dependencies": dependencies,
        },
        type = "go_binary",
    )