package starlarkobject

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func ExampleObject() {
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
	__init__,
	test,
	other_func,
	str="hello"
)
obj = SimpleObject("shahar")
obj.other_func()

WithSuperClass = object("WithSuperClass", SimpleObject)
s = WithSuperClass()
s.test()

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
	// hello
	// world
	// shahar
	// world
	// shahar
	//
	// Globals:
	// SimpleObject (object) = ()
	// WithSuperClass (object) = ()
	// __init__ (function) = <function __init__>
	// obj () = ([__init__ new_attr other_func str test])
	// other_func (function) = <function other_func>
	// s () = ([__init__ new_attr other_func str test])
	// test (function) = <function test>
}

func starlarkHelper(input, filename string) ([]string, starlark.StringDict, error) {
	var output []string
	thread := &starlark.Thread{
		Name: "example",
		Print: func(_ *starlark.Thread, msg string) {
			output = append(output, msg)
		},
	}

	predeclared := starlark.StringDict{
		"object": starlark.NewBuiltin("object", MakeObject),
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
	}

	// Execute a program.
	globals, err := starlark.ExecFile(thread, filename, input, predeclared)
	return output, globals, err
}

func TestObject_Attr(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    []string
		wantErr bool
	}{
		{
			name: "get_attr.star",
			code: `
MyClass = object("MyClass", attr=0)
obj = MyClass()
print(obj.attr)
			`,
			want:    []string{"0"},
			wantErr: false,
		},
		{
			name: "get_attr_super.star",
			code: `
MyClass = object("MyClass", attr=0)
MySubClass = object("MySubClass", MyClass)
obj=MySubClass()
print(obj.attr)
			`,
			want:    []string{"0"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := starlarkHelper(tt.code, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Object.Attr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Object.Attr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestObject_SetField(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    []string
		wantErr bool
	}{
		{
			name: "reset_field.star",
			code: `
MyClass = object("MyClass", attr=0)
obj = MyClass()
obj.attr=1
print(obj.attr)
			`,
			want:    []string{"1"},
			wantErr: false,
		},
		{
			name: "reset_super_field.star",
			code: `
def print_attr(self):
	print(self.attr)
MyClass = object("MyClass", print_attr, attr=0)
MySubClass = object("MySubClass", MyClass)
obj=MySubClass()
obj.attr=1
obj.print_attr()
			`,
			want:    []string{"1"},
			wantErr: false,
		},
		{
			name: "set_unknown_field.star",
			code: `
MyClass = object("MyClass")
obj=MyClass()
obj.attr=1
print(obj.attr)
			`,
			want:    []string{"1"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := starlarkHelper(tt.code, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Object.Attr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Object.Attr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestObject_String(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    []string
		wantErr bool
	}{
		{
			name: "str_formatting.star",
			code: `
def __str__(self):
	return self.attr

MyClass = object("MyClass", __str__, attr=0)
print(MyClass())
			`,
			want:    []string{`0`},
			wantErr: false,
		},
		{
			name: "str_formatting_calling_arg_from_super.star",
			code: `
def __str__(self):
	return self.attr2

MyClass = object("MyClass", attr=0, attr2=1)
MySubClass = object("MySubClass", __str__, MyClass)
print(MySubClass())
					`,
			want:    []string{`1`},
			wantErr: false,
		},
		{
			name: "super_str.star",
			code: `
def __str__(self):
	return self.attr2

MyClass = object("MyClass", __str__, attr=0, attr2=1)
MySubClass = object("MySubClass", MyClass)
print(MySubClass())
					`,
			want:    []string{`1`},
			wantErr: false,
		},
		{
			name: "str_is_not_a_function.star",
			code: `

MyClass = object("MyClass", __str__="hello")
print(MyClass())
					`,
			want:    []string{`([__str__])`},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := starlarkHelper(tt.code, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Object.Attr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Object.Attr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestObject_Truth(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    []string
		wantErr bool
	}{
		{
			name: "bool_is_true.star",
			code: `
def __bool__(self):
	return True

MyClass = object("MyClass", __bool__)
print("x") if MyClass() else print("y")
			`,
			want:    []string{`x`},
			wantErr: false,
		},
		{
			name: "bool_is_false.star",
			code: `
def __bool__(self):
	return False

MyClass = object("MyClass", __bool__)
print("x") if MyClass() else print("y")
			`,
			want:    []string{`y`},
			wantErr: false,
		},
		{
			name: "bool_from_supper.star",
			code: `
def __bool__(self):
	return False 

MyClass = object("MyClass", __bool__)
MySubClass = object("MySubClass", MyClass)
print("x") if MySubClass() else print("y")
			`,
			want:    []string{`y`},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := starlarkHelper(tt.code, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Object.Attr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Object.Attr() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestObject_Struct(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    []string
		wantErr bool
	}{
		{
			name: "extending_struct.star",
			code: `
Item = struct(attr="x")
MyClass = object("MyClass", Item)
obj = MyClass()
print(obj.attr)
			`,
			want:    []string{`x`},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := starlarkHelper(tt.code, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Object.Attr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Object.Attr() = %v, want %v", got, tt.want)
			}
		})
	}
}
