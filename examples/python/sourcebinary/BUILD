load("std/python", "py_source_binary")
load("examples/python/sourcelibrary", sourcelibrary = "lib")
load("3rdParty/python", "requests")

sourcebinary = py_source_binary(
    name = "sourcebinary",
    package_name = "sourcebinary",
    sources = glob(
        "setup.py",
        "src/sourcebinary/main.py",
        "src/sourcebinary/__init__.py",
    ),
    dependencies = [
        sourcelibrary,
        requests,
    ],
    entry_point = "main",
)
