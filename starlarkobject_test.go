package starlarkobject

import (
	"fmt"
	"log"

	"go.starlark.net/starlark"
)

func ExampleStarlarkObject() {
	const data = `

def __init__(self, new_attr="hi"):
	"""initial doc"""
	self.new_attr=new_attr

def other_func(self):
	print(self.str)
	self.str="world"
	self.test()

def test(self):
	print(self.str)
	print(self.new_attr)

SimpleObject = object("SimpleObject",
	__init__=__init__,
	test=test,
	other_func=other_func,
	str="hello"
)
obj = SimpleObject("shahar")
obj.other_func()
`

	thread := &starlark.Thread{
		Name: "example",
		Print: func(_ *starlark.Thread, msg string) {
			fmt.Println(msg)
		},
	}

	predeclared := starlark.StringDict{
		"object": starlark.NewBuiltin("object", MakeObject),
	}

	// Execute a program.
	globals, err := starlark.ExecFile(thread, "apparent/filename.star", data, predeclared)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			log.Fatal(evalErr.Backtrace())
		}
		log.Fatal(err)
	}

	// Print the global environment.
	fmt.Println("\nGlobals:")
	for _, name := range globals.Keys() {
		v := globals[name]
		fmt.Printf("%s (%s) = %s\n", name, v.Type(), v.String())
	}
	// Output:
	// Globals:
	// obj (object) = obj(hello)
}
