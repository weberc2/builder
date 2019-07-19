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

six = pypi(name = "six")
python_dateutil = pypi(name = "python-dateutil", dependencies = [six])
pytz = pypi(name = "pytz")
numpy = pypi(name = "numpy")
pandas = pypi(name = "pandas", dependencies = [python_dateutil, pytz, numpy])

aenum = pypi(name = "aenum")
attrs = pypi(name = "attrs")
newrelic = pypi(name = "newrelic")
arrow = pypi(
    name = "arrow",
    constraint = "==0.12.1",
    dependencies = [python_dateutil],
)
sentry_sdk = pypi(name = "sentry-sdk", dependencies = [certifi, urllib3])
async_timeout = pypi(name = "async-timeout")
hiredis = pypi(name = "hiredis")
aioredis = pypi(name = "aioredis")
sentinels = pypi(name = "sentinels")
sqlalchemy = pypi(name = "sqlalchemy")
slugify = pypi(name = "slugify")
astrolabe = pypi(name = "astrolabe")
multidict = pypi(name = "multidict")
yarl = pypi(name = "yarl", dependencies = [idna])
idna_ssl = pypi(name = "idna-ssl")
typing_extensions = pypi(name = "typing_extensions")
aiohttp = pypi(
    name = "aiohttp",
    constraint = "==3.4.4",
    dependencies = [
        attrs,
        chardet,
        multidict,
        async_timeout,
        yarl,
        idna_ssl,
        typing_extensions,
    ],
)
isoweek = pypi(name = "isoweek")