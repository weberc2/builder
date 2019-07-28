package lib

import "fmt"

type Person struct {
	Name string
	Age  int
}

func GreetPerson(p Person) {
	fmt.Printf("Hello, %s!\n", p.Name)
}
