// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)


// MAASModel represents a model object as returned by the MAAS API.  This is
// a special kind of MAASObject.  A MAAS API call will usually return either
// a MAASModel or a list of MAASModels.  (The list itself will be wrapped in
// a MAASObject).
type MAASModel interface {
	// Resource URI for this object.
	URL() string
	// Retrieve this model object.
	Get() (MAASModel, error)
	// Write this model object.
	Post(params url.Values) (MAASModel, error)
	// Update this model object with the given values.
	Update(params url.Values) (MAASModel, error)
	// Delete this model object.
	Delete() error
	// Invoke a GET-based method on this model object.
	CallGet(operation string, params url.Values) (MAASObject, error)
	// Invoke a POST-based method on this model object.
	CallPost(operation string, params url.Values) (MAASObject, error)
}


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
// There is one exception: a MAASModel is really a special kind of map,
// so you can read it as either.
// Reading a null item is also an error.  So before you try obj.Get*(),
// first check that obj != nil.
type MAASObject interface {
	// Type of this value:
	// "string", "float64", "map", "model", "array", or "bool".
	Type() string
	// Read as string.
	GetString() (string, error)
	// Read number as float64.
	GetFloat64() (float64, error)
	// Read object as map.
	GetMap() (map[string]MAASObject, error)
	// Read object as MAAS model object.
	GetModel() (MAASModel, error)
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
// One type is special: maasModel is a model object.  It behaves just like
// a maasMap if you want it to, but it also implements MAASModel.
type maasString string
type maasFloat64 float64
type maasMap map[string]MAASObject
type maasModel maasMap
type maasArray []MAASObject
type maasBool bool


const resource_uri = "resource_uri"

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
		if _, ok := result[resource_uri]; ok {
			// If the map contains "resource-uri", we can treat
			// it as a model object.
			return maasModel(result)
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
	msg := fmt.Sprintf("Unknown JSON type, can't be converted to MAASObject: %v", value)
	panic(msg)
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


// Error return values for failure to convert to string.
func failString(obj MAASObject) (string, error) {
	return "", failConversion("string", obj)
}
// Error return values for failure to convert to float64.
func failFloat64(obj MAASObject) (float64, error) {
	return 0.0, failConversion("float64", obj)
}
// Error return values for failure to convert to map.
func failMap(obj MAASObject) (map[string]MAASObject, error) {
	return make(map[string]MAASObject, 0), failConversion("map", obj)
}
// Error return values for failure to convert to model.
func failModel(obj MAASObject) (MAASModel, error) {
	return maasModel{}, failConversion("model", obj)
}
// Error return values for failure to convert to array.
func failArray(obj MAASObject) ([]MAASObject, error) {
	return make([]MAASObject, 0), failConversion("array", obj)
}
// Error return values for failure to convert to bool.
func failBool(obj MAASObject) (bool, error) {
	return false, failConversion("bool", obj)
}


// MAASObject implementation for maasString.
func (maasString) Type() string { return "string" }
func (obj maasString) GetString() (string, error) { return string(obj), nil }
func (obj maasString) GetFloat64() (float64, error) { return failFloat64(obj) }
func (obj maasString) GetMap() (map[string]MAASObject, error) { return failMap(obj) }
func (obj maasString) GetModel() (MAASModel, error) { return failModel(obj) }
func (obj maasString) GetArray() ([]MAASObject, error) { return failArray(obj) }
func (obj maasString) GetBool() (bool, error) { return failBool(obj) }

// MAASObject implementation for maasFloat64.
func (maasFloat64) Type() string { return "float64" }
func (obj maasFloat64) GetString() (string, error) { return failString(obj) }
func (obj maasFloat64) GetFloat64() (float64, error) { return float64(obj), nil }
func (obj maasFloat64) GetMap() (map[string]MAASObject, error) { return failMap(obj) }
func (obj maasFloat64) GetModel() (MAASModel, error) { return failModel(obj) }
func (obj maasFloat64) GetArray() ([]MAASObject, error) { return failArray(obj) }
func (obj maasFloat64) GetBool() (bool, error) { return failBool(obj) }

// MAASObject implementation for maasMap.
func (maasMap) Type() string { return "map" }
func (obj maasMap) GetString() (string, error) { return failString(obj) }
func (obj maasMap) GetFloat64() (float64, error) { return failFloat64(obj) }
func (obj maasMap) GetMap() (map[string]MAASObject, error) {
	return (map[string]MAASObject)(obj), nil
}
func (obj maasMap) GetModel() (MAASModel, error) { return failModel(obj) }
func (obj maasMap) GetArray() ([]MAASObject, error) { return failArray(obj) }
func (obj maasMap) GetBool() (bool, error) { return failBool(obj) }


// MAASObject implementation for maasModel.
func (maasModel) Type() string { return "model" }
func (obj maasModel) GetString() (string, error) { return failString(obj) }
func (obj maasModel) GetFloat64() (float64, error) { return failFloat64(obj) }
func (obj maasModel) GetMap() (map[string]MAASObject, error) {
	return (map[string]MAASObject)(obj), nil
}
func (obj maasModel) GetModel() (MAASModel, error) {
	return maasModel(obj), nil
}
func (obj maasModel) GetArray() ([]MAASObject, error) { return failArray(obj) }
func (obj maasModel) GetBool() (bool, error) { return failBool(obj) }


// MAASModel implementation for maasModel.

func (obj maasModel) URL() string {
	contents, err := obj.GetMap()
	if err != nil {
		panic("Unexpected failure converting maasModel to maasMap.")
	}
	url, err := contents[resource_uri].GetString()
	if err != nil {
		panic("Unexpected failure reading maasModel's URL.")
	}
	return url
}

var NotImplemented = errors.New("Not implemented")

func (obj maasModel) Get() (MAASModel, error) {
	return maasModel{}, NotImplemented
}

func (obj maasModel) Post(params url.Values) (MAASModel, error) {
	return maasModel{}, NotImplemented
}

func (obj maasModel) Update(params url.Values) (MAASModel, error) {
	return maasModel{}, NotImplemented
}

func (obj maasModel) Delete() error { return NotImplemented }

func (obj maasModel) CallGet(operation string, params url.Values) (MAASObject, error) {
	return nil, NotImplemented
}
func (obj maasModel) CallPost(operation string, params url.Values) (MAASObject, error) {
	return nil, NotImplemented
}


// MAASObject implementation for maasArray.
func (maasArray) Type() string { return "array" }
func (obj maasArray) GetString() (string, error) { return failString(obj) }
func (obj maasArray) GetFloat64() (float64, error) { return failFloat64(obj) }
func (obj maasArray) GetMap() (map[string]MAASObject, error) { return failMap(obj) }
func (obj maasArray) GetModel() (MAASModel, error) { return failModel(obj) }
func (obj maasArray) GetArray() ([]MAASObject, error) {
	return ([]MAASObject)(obj), nil
}
func (obj maasArray) GetBool() (bool, error) { return failBool(obj) }

// MAASObject implementation for maasBool.
func (maasBool) Type() string { return "bool" }
func (obj maasBool) GetString() (string, error) { return failString(obj) }
func (obj maasBool) GetFloat64() (float64, error) { return failFloat64(obj) }
func (obj maasBool) GetMap() (map[string]MAASObject, error) { return failMap(obj) }
func (obj maasBool) GetModel() (MAASModel, error) { return failModel(obj) }
func (obj maasBool) GetArray() ([]MAASObject, error) { return failArray(obj) }
func (obj maasBool) GetBool() (bool, error) { return bool(obj), nil }
