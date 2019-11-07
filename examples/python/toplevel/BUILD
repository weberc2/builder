load("examples/python/aio_utils", aio_utils = "unit")
load("examples/python/application_config", application_config = "unit")
load("examples/python/connector_service", connector_service = "unit")
load("examples/python/data_fields", data_fields = "unit")
load("examples/python/emails", emails = "unit")
load("examples/python/feature_flag_client", feature_flag_client = "unit")
load("examples/python/ideas", ideas = "unit")
load("examples/python/ml", ml = "unit")
load("examples/python/nlg_engine", nlg_engine = "unit")
load("examples/python/ontology", ontology = "unit")
load("examples/python/python_core", python_core = "unit")
load("examples/python/python_logger", python_logger = "unit")
load("examples/python/story_utils", story_utils = "unit")
load("examples/python/time_utils", time_utils = "unit")

# This is a meta-target; it gives us an easy way to run all unit tests:
# builder build examples/python/toplevel:unit
unit = mktarget(
    type = "noop",
    name = "unit",
    args = {
        "dependencies": [
            aio_utils,
            application_config,
            connector_service,
            data_fields,
            feature_flag_client,
            ideas,
            ml,
            nlg_engine,
            ontology,
            # python_core unit tests are currently broken due to a circular
            # dependency with data_fields
            # python_core,
            python_logger,
            story_utils,
            time_utils,
        ],
    },
)
