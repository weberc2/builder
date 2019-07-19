package plugins

import (
	"github.com/weberc2/builder/plugins/golang"
	"github.com/weberc2/builder/plugins/python"
)

var GoBinary = golang.Binary
var PySourceBinary = python.SourceBinary
var PySourceLibrary = python.SourceLibrary
var PyPypiLibrary = python.PypiLibrary
var PyTest = python.Test
