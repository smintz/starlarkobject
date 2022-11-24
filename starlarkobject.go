package starlarkobject

import (
	"fmt"

	"go.starlark.net/starlark"
)

type Object struct {
	Name    string
	Members starlark.StringDict
	Super   *Super
}

var _ starlark.Value = (*Object)(nil)

func (o *Object) String() string {
	return fmt.Sprintf("%s(%v)", o.Name, o.AttrNames())
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
func (o *Object) Attr(name string) (starlark.Value, error) {
	if value, ok := o.Members[name]; ok {
		return value, nil
	}
	return o.Super.Attr(name)
}
func (o *Object) AttrNames() []string { return append(o.Members.Keys(), o.Super.AttrNames()...) }
func (o *Object) SetField(name string, val starlark.Value) error {

	o.Members[name] = val
	return nil
}

func (o *Object) Freeze() { o.Members.Freeze() }

func MakeObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string

	_ = starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &name)

	obj := &Object{Name: name, Super: &Super{}}
	members := make(starlark.StringDict, len(kwargs))

	for _, item := range args[1:] {
		if f, ok := item.(*starlark.Function); ok {
			members[f.Name()] = &function{object: obj, name: f.Name(), function: f}
			continue
		}
		if callable, ok := item.(starlark.Callable); ok {
			callableItem, err := callable.CallInternal(thread, args, kwargs)
			if err != nil {
				return nil, err
			}
			obj.Super = &Super{value: callableItem, super: obj.Super}
		}
	}
	for _, kwarg := range kwargs {
		k := string(kwarg[0].(starlark.String))
		f, ok := kwarg[1].(*starlark.Function)
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

type Super struct {
	value starlark.Value
	super starlark.Value
}

func (s *Super) String() string {
	var returnValue string
	s.valueOrSuper(func(v starlark.Value) error {
		returnValue = v.String()
		return nil
	})
	return returnValue
}
func (s *Super) Type() string {
	var returnValue string
	s.valueOrSuper(func(v starlark.Value) error {
		returnValue = v.Type()
		return nil
	})
	return returnValue
}
func (s *Super) Truth() starlark.Bool {
	var returnValue starlark.Bool
	s.valueOrSuper(
		func(v starlark.Value) error {
			returnValue = v.Truth()
			return nil
		},
	)
	return returnValue
}

func (s *Super) Freeze() {
	s.value.Freeze()
	s.super.Freeze()
}

func (s *Super) Hash() (uint32, error) {
	var returnValue uint32
	var err error
	err = s.valueOrSuper(func(v starlark.Value) error {
		returnValue, err = v.Hash()
		return err
	})
	return returnValue, err
}

func (s *Super) valueOrSuper(f func(starlark.Value) error) error {
	err := f(s.value)
	if err == nil {
		return nil
	}
	if s.super == nil {
		return nil
	}
	return f(s.super)
}

func (s *Super) valueThenSuper(f func(starlark.Value) error) error {
	if s.value == nil {
		return nil
	}
	err := f(s.value)
	if err != nil {
		return err
	}
	if s.super == nil {
		return nil
	}
	return f(s.super)
}

func (s *Super) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var returnValue starlark.Value
	err := s.valueOrSuper(func(v starlark.Value) error {
		var e error
		if val, ok := v.(starlark.Callable); ok {
			returnValue, e = val.CallInternal(thread, args, kwargs)
		}
		return e
	})
	return returnValue, err
}

func (s *Super) AttrNames() []string {
	var returnValue []string
	s.valueThenSuper(func(v starlark.Value) error {
		if v == nil {
			return fmt.Errorf("got nil")
		}
		if value, ok := v.(starlark.HasAttrs); ok {
			returnValue = append(returnValue, value.AttrNames()...)
		}
		return nil
	})
	return returnValue
}

func (s *Super) Attr(name string) (starlark.Value, error) {
	var returnValue starlark.Value
	err := s.valueOrSuper(func(v starlark.Value) error {
		e := fmt.Errorf("cannot find field %s", name)
		if v, ok := s.value.(starlark.HasAttrs); ok {
			returnValue, e = v.Attr(name)
		}
		return e
	})
	return returnValue, err
}

func (s *Super) SetField(name string, val starlark.Value) error {
	return s.valueOrSuper(func(v starlark.Value) error {
		if obj, ok := v.(starlark.HasSetField); ok {
			return obj.SetField(name, val)
		}
		return fmt.Errorf("SetField not implemented for %s", v.Type())
	})
}
