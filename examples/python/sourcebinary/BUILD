load("py", "py_source_binary")

py_source_binary(
    name = "sourcebinary",
    package_name = "sourcebinary",
    sources = file(
        "setup.py",
        "src/sourcebinary/main.py",
        "src/sourcebinary/__init__.py",
    ),
    dependencies = [
        reftarget("//examples/python/sourcelibrary:sourcelibrary"),
        reftarget("//3rdParty/python:requests"),
    ],
    entry_point = "main",
)