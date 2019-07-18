# This probably won't work if there are dependencies on local packages since
# we're using modules; will need to put local dependencies in a GOPATH somehow.

def go_binary(name, package_name, sources, local_dependencies = None):
    args = {
        "package_name": package_name,
        "sources": sources,
    }
    if local_dependencies != None:
        args["local_dependencies"] = local_dependencies
    return mktarget(name = name, args = args, type = "go_binary")