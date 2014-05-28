package otto

import (
	"fmt"
)

// ======
// _stash
// ======

type _stash interface {
	hasBinding(string) bool            //
	createBinding(string, bool, Value) // CreateMutableBinding
	setBinding(string, Value, bool)    // SetMutableBinding
	getBinding(string, bool) Value     // GetBindingValue
	deleteBinding(string) bool         //
	setValue(string, Value, bool)      // createBinding + setBinding

	outer() _stash
	runtime() *_runtime

	newReference(string, bool) _reference

	clone(clone *_clone) _stash
}

// ==========
// _objectStash
// ==========

type _objectStash struct {
	_runtime *_runtime
	_outer   _stash
	object   *_object
}

func (self *_objectStash) runtime() *_runtime {
	return self._runtime
}

func (runtime *_runtime) newObjectStash(object *_object, outer _stash) *_objectStash {
	if object == nil {
		object = runtime.newBaseObject()
		object.class = "environment"
	}
	return &_objectStash{
		_runtime: runtime,
		_outer:   outer,
		object:   object,
	}
}

func (self0 *_objectStash) clone(clone *_clone) _stash {
	self1, exists := clone.objectStash(self0)
	if exists {
		return self1
	}
	*self1 = _objectStash{
		clone.runtime,
		clone.stash(self0._outer),
		clone.object(self0.object),
	}
	return self1
}

func (self *_objectStash) hasBinding(name string) bool {
	return self.object.hasProperty(name)
}

func (self *_objectStash) createBinding(name string, deletable bool, value Value) {
	if self.object.hasProperty(name) {
		panic(hereBeDragons())
	}
	mode := _propertyMode(0111)
	if !deletable {
		mode = _propertyMode(0110)
	}
	// TODO False?
	self.object.defineProperty(name, value, mode, false)
}

func (self *_objectStash) setBinding(name string, value Value, strict bool) {
	self.object.put(name, value, strict)
}

func (self *_objectStash) setValue(name string, value Value, throw bool) {
	if !self.hasBinding(name) {
		self.createBinding(name, true, value) // Configurable by default
	} else {
		self.setBinding(name, value, throw)
	}
}

func (self *_objectStash) getBinding(name string, throw bool) Value {
	if self.object.hasProperty(name) {
		return self.object.get(name)
	}
	if throw { // strict?
		panic(newReferenceError("Not Defined", name))
	}
	return UndefinedValue()
}

func (self *_objectStash) deleteBinding(name string) bool {
	return self.object.delete(name, false)
}

func (self *_objectStash) outer() _stash {
	return self._outer
}

func (self *_objectStash) newReference(name string, strict bool) _reference {
	return newPropertyReference(self.object, name, strict)
}

// =========
// _dclStash
// =========

type _dclStash struct {
	_runtime *_runtime
	_outer   _stash
	property map[string]_dclProperty
}

type _dclProperty struct {
	value     Value
	mutable   bool
	deletable bool
	readable  bool
}

func (runtime *_runtime) newDeclarationStash(outer _stash) *_dclStash {
	return &_dclStash{
		_runtime: runtime,
		_outer:   outer,
		property: map[string]_dclProperty{},
	}
}

func (self0 *_dclStash) clone(clone *_clone) _stash {
	self1, exists := clone.dclStash(self0)
	if exists {
		return self1
	}
	property := make(map[string]_dclProperty, len(self0.property))
	for index, value := range self0.property {
		property[index] = clone.dclProperty(value)
	}
	*self1 = _dclStash{
		clone.runtime,
		clone.stash(self0._outer),
		property,
	}
	return self1
}

func (self *_dclStash) hasBinding(name string) bool {
	_, exists := self.property[name]
	return exists
}

func (self *_dclStash) runtime() *_runtime {
	return self._runtime
}

func (self *_dclStash) createBinding(name string, deletable bool, value Value) {
	_, exists := self.property[name]
	if exists {
		panic(fmt.Errorf("createBinding: %s: already exists", name))
	}
	self.property[name] = _dclProperty{
		value:     value,
		mutable:   true,
		deletable: deletable,
		readable:  false,
	}
}

func (self *_dclStash) setBinding(name string, value Value, strict bool) {
	property, exists := self.property[name]
	if !exists {
		panic(fmt.Errorf("setBinding: %s: missing", name))
	}
	if property.mutable {
		property.value = value
		self.property[name] = property
	} else {
		typeErrorResult(strict)
	}
}

func (self *_dclStash) setValue(name string, value Value, throw bool) {
	if !self.hasBinding(name) {
		self.createBinding(name, false, value) // NOT deletable by default
	} else {
		self.setBinding(name, value, throw)
	}
}

// FIXME This is called a __lot__
func (self *_dclStash) getBinding(name string, throw bool) Value {
	property, exists := self.property[name]
	if !exists {
		panic(fmt.Errorf("getBinding: %s: missing", name))
	}
	if !property.mutable && !property.readable {
		if throw { // strict?
			panic(newTypeError())
		}
		return UndefinedValue()
	}
	return property.value
}

func (self *_dclStash) deleteBinding(name string) bool {
	property, exists := self.property[name]
	if !exists {
		return true
	}
	if !property.deletable {
		return false
	}
	delete(self.property, name)
	return true
}

func (self *_dclStash) outer() _stash {
	return self._outer
}

func (self *_dclStash) newReference(name string, strict bool) _reference {
	return &_stashReference{
		name: name,
		base: self,
	}
}

// ========
// _fnStash
// ========

type _fnStash struct {
	_dclStash
	arguments           *_object
	indexOfArgumentName map[string]string
}

func (runtime *_runtime) newFunctionStash(outer _stash) *_fnStash {
	return &_fnStash{
		_dclStash: _dclStash{
			_runtime: runtime,
			_outer:   outer,
			property: map[string]_dclProperty{},
		},
	}
}

func (self0 _fnStash) clone(clone *_clone) _stash {
	// FIXME Memory leak issue here?
	return &_fnStash{
		*(self0._dclStash.clone(clone).(*_dclStash)),
		clone.object(self0.arguments),
		self0.indexOfArgumentName,
	}
}
