// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"errors"
	"fmt"
)


// MAASObject is a wrapper around a JSON structure which provides
// methods to extract data from that structure.
// A MAASObject provides a simple structure consisting of the data types
// defined in JSON: string, number, object, list, and bool.  To get the
// value you want out of a MAASObject, you must know (or figure out) which
// kind of value you have, and then call the appropriate Get*() method to
// get at it.  Reading an item as the wrong type will return an error.
// For instance, if your MAASObject consists of a number, call GetFloat64()
// to get the value as a float64.  If it's a list, call GetArray() to get
// a slice of MAASObjects.  To read any given item from the slice, you'll
// need to "Get" that as the right type as well.
// Reading a null item is also an error.  So before you try obj.Get*(),
// first check that obj != nil.
type MAASObject interface {
	// Type of this value: "string", "float64", "map", "array", or "bool".
	Type() string
	// Read as string.
	GetString() (string, error)
	// Read number as float64.
	GetFloat64() (float64, error)
	// Read object as map.
	GetMap() (map[string]MAASObject, error)
	// Read list as array.
	GetArray() ([]MAASObject, error)
	// Read as bool.
	GetBool() (bool, error)
}


// Internally, each MAASObject already knows what type it is.  It just
// can't tell the caller yet because the caller may not have the right
// hard-coded variable type.
// So for each JSON type, there is a separate implementation of MAASObject
// that converts only to that type.  Any other conversion is an error.
type maasString string
type maasFloat64 float64
type maasMap map[string]MAASObject
type maasArray []MAASObject
type maasBool bool


// Internal: turn a completely untyped json.Unmarshal result into a
// MAASObject (with the appropriate implementation of course).
// This function is recursive.  Maps and arrays are deep-copied, with each
// individual value being converted to a MAASObject type.
func maasify(value interface{}) MAASObject {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case string:
		return maasString(value.(string))
	case float64:
		return maasFloat64(value.(float64))
	case map[string]interface{}:
		original := value.(map[string]interface{})
		result := make(map[string]MAASObject, len(original))
		for key, value := range original {
			result[key] = maasify(value)
		}
		return maasMap(result)
	case []interface{}:
		original := value.([]interface{})
		result := make([]MAASObject, len(original))
		for index, value := range original {
			result[index] = maasify(value)
		}
		return maasArray(result)
	case bool:
		return maasBool(value.(bool))
	}
	panic(fmt.Sprintf("Unknown JSON type, can't be converted to MAASObject: %v", value))
}


// Parse a JSON blob into a MAASObject.
func Parse(input []byte) (MAASObject, error) {
	var obj interface{}
	err := json.Unmarshal(input, &obj)
	if err != nil {
		return nil, err
	}
	return maasify(obj), nil
}


// Return error value for failed type conversion.
func failConversion(wanted_type string, obj MAASObject) error {
	msg := fmt.Sprintf("Requested %v, got %v.", wanted_type, obj.Type())
	return errors.New(msg)
}


// Make an empty map for returning with an error.
func blankMap() map[string]MAASObject {
	return make(map[string]MAASObject, 0)
}


// Make an empty array for returning with an error.
func blankArray() []MAASObject {
	return make([]MAASObject, 0)
}


// Implementations of MAASObject types follow.

func (maasString) Type() string { return "string" }
func (obj maasString) GetString() (string, error) { return string(obj), nil }
func (obj maasString) GetFloat64() (float64, error) {
	return 0, failConversion("float64", obj)
}
func (obj maasString) GetMap() (map[string]MAASObject, error) {
	return blankMap(), failConversion("map", obj)
}
func (obj maasString) GetArray() ([]MAASObject, error) {
	return blankArray(), failConversion("map", obj)
}
func (obj maasString) GetBool() (bool, error) {
	return false, failConversion("bool", obj)
}

func (maasFloat64) Type() string { return "float64" }
func (obj maasFloat64) GetString() (string, error) {
	return "", failConversion("string", obj)
}
func (obj maasFloat64) GetFloat64() (float64, error) { return float64(obj), nil }
func (obj maasFloat64) GetMap() (map[string]MAASObject, error) {
	return blankMap(), failConversion("map", obj)
}
func (obj maasFloat64) GetArray() ([]MAASObject, error) {
	return blankArray(), failConversion("map", obj)
}
func (obj maasFloat64) GetBool() (bool, error) {
	return false, failConversion("bool", obj)
}

func (maasMap) Type() string { return "map" }
func (obj maasMap) GetString() (string, error) {
	return "", failConversion("string", obj)
}
func (obj maasMap) GetFloat64() (float64, error) {
	return 0.0, failConversion("float64", obj)
}
func (obj maasMap) GetMap() (map[string]MAASObject, error) {
	return (map[string]MAASObject)(obj), nil
}
func (obj maasMap) GetArray() ([]MAASObject, error) {
	return blankArray(), failConversion("map", obj)
}
func (obj maasMap) GetBool() (bool, error) {
	return false, failConversion("bool", obj)
}

func (maasArray) Type() string { return "array" }
func (obj maasArray) GetString() (string, error) {
	return "", failConversion("string", obj)
}
func (obj maasArray) GetFloat64() (float64, error) {
	return 0.0, failConversion("float64", obj)
}
func (obj maasArray) GetMap() (map[string]MAASObject, error) {
	return blankMap(), failConversion("map", obj)
}
func (obj maasArray) GetArray() ([]MAASObject, error) {
	return ([]MAASObject)(obj), nil
}
func (obj maasArray) GetBool() (bool, error) {
	return false, failConversion("bool", obj)
}

func (maasBool) Type() string { return "bool" }
func (obj maasBool) GetString() (string, error) {
	return "", failConversion("string", obj)
}
func (obj maasBool) GetFloat64() (float64, error) {
	return 0.0, failConversion("float64", obj)
}
func (obj maasBool) GetMap() (map[string]MAASObject, error) {
	return blankMap(), failConversion("map", obj)
}
func (obj maasBool) GetArray() ([]MAASObject, error) {
	return blankArray(), failConversion("map", obj)
}
func (obj maasBool) GetBool() (bool, error) {
	return bool(obj), nil
}

