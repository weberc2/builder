package golang

const BuiltinModule = `
load("std/command", "bash")

def go_module(name, sources, directory = None):
	return bash(
		name = name,
		environment = {
			"SOURCES": sources,
			"DIRECTORY": directory if directory != None else ""
		},
		script = 'cd "$SOURCES/$DIRECTORY" && CGO_ENABLED=0 go build -o "$OUTPUT"',
	)
`
