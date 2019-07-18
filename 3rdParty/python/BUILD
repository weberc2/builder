load("py", pypi="py_pypi_library")

certifi = pypi(name = "certifi")
chardet = pypi(name = "chardet")
idna = pypi(name = "idna")
urllib3 = pypi(name = "urllib3")
requests = pypi(
    name = "requests",
    dependencies = [certifi, chardet, idna, urllib3],
)

psycopg2_binary = pypi(name = "psycopg2-binary")
aiopg = pypi(name = "aiopg", dependencies = [psycopg2_binary])
aiotask_context = pypi(name = "aiotask_context")