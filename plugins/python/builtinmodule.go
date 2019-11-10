package python

const BuiltinModule = `
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

def pypi(
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
        type = "pypi",
    )

def virtualenv(name, dependencies):
    return mktarget(
        name = name,
        args = {"dependencies": dependencies},
        type = "virtualenv",
    )

atomicwrites = pypi(name = "atomicwrites")
attrs = pypi(name = "attrs")
packaging = pypi(name = "packaging")
six = pypi(name = "six")
more_itertools = pypi(name = "more-itertools", dependencies = [ six ])
zipp = pypi(name = "zipp", dependencies = [ more_itertools ])
importlib_metadata = pypi(name = "importlib_metadata", dependencies = [ zipp ])
pluggy = pypi(name = "pluggy", dependencies = [ importlib_metadata ])
py = pypi(name = "py")
wcwidth = pypi(name = "wcwidth")

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
		type = "pytest",
	)
`
