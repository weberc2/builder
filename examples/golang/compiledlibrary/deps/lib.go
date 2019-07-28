package deps

import (
	lib "github.com/weberc2/builder/examples/golang/compiledlibrary/nodeps"
)

func GreetSomeone(name string) {
	lib.GreetPerson(lib.Person{Name: name})
}
