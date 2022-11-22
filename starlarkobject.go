package starlarkobject

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

type Object struct {
	Name    string
	Members starlark.StringDict
}

var _ starlark.Value = (*Object)(nil)

func (o *Object) String() string {
	return fmt.Sprintf("%s(%s)", o.Name, strings.Join(o.AttrNames(), ", "))
}

func (o *Object) Type() string {
	return o.Name
}

func (o *Object) Truth() starlark.Bool {
	return starlark.False
}

func (o *Object) Hash() (uint32, error) {
	return 0, fmt.Errorf("not hashable")
}
func (o *Object) Attr(name string) (starlark.Value, error) { return o.Members[name], nil }
func (o *Object) AttrNames() []string                      { return o.Members.Keys() }
func (o *Object) SetField(name string, val starlark.Value) error {
	o.Members[name] = val
	return nil
}

func (o *Object) Freeze() { o.Members.Freeze() }

func MakeObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, nil, 1, &name); err != nil {
		return nil, err
	}
	obj := &Object{Name: name}
	members := make(starlark.StringDict, len(kwargs))
	for _, kwarg := range kwargs {
		k := string(kwarg[0].(starlark.String))
		f, ok := kwarg[1].(starlark.Callable)
		if ok {
			members[k] = &function{object: obj, name: k, function: f}
		} else {
			members[k] = kwarg[1]
		}
	}
	obj.Members = members
	return &objectInit{obj}, nil
}

type function struct {
	object   *Object
	name     string
	function starlark.Callable
}

var _ starlark.Callable = (*function)(nil)

func (fun *function) Name() string          { return fun.name }
func (fun *function) String() string        { return fun.name }
func (fun *function) Type() string          { return "symbol" }
func (fun *function) Freeze()               {} // immutable
func (fun *function) Truth() starlark.Bool  { return starlark.True }
func (fun *function) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", fun.Type()) }

func (fun *function) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	args = append(starlark.Tuple{
		fun.object,
	}, args...)
	return fun.function.CallInternal(thread, args, kwargs)
}

type objectInit struct {
	object *Object
}

var _ starlark.Callable = (*function)(nil)

func (ob *objectInit) Name() string          { return ob.object.Name }
func (ob *objectInit) String() string        { return ob.object.Name + "()" }
func (ob *objectInit) Type() string          { return "object" }
func (ob *objectInit) Freeze()               {} // immutable
func (ob *objectInit) Truth() starlark.Bool  { return starlark.True }
func (ob *objectInit) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", ob.Type()) }

func (ob *objectInit) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if initFunc, ok := ob.object.Members["__init__"]; ok {
		if f, isFunc := initFunc.(*function); isFunc {
			f.CallInternal(thread, args, kwargs)
		}
	}
	return ob.object, nil
}
