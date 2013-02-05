// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"errors"
	"fmt"
)

// JSONObject is a wrapper around a JSON structure which provides
// methods to extract data from that structure.
// A JSONObject provides a simple structure consisting of the data types
// defined in JSON: string, number, object, list, and bool.  To get the
// value you want out of a JSONObject, you must know (or figure out) which
// kind of value you have, and then call the appropriate Get*() method to
// get at it.  Reading an item as the wrong type will return an error.
// For instance, if your JSONObject consists of a number, call GetFloat64()
// to get the value as a float64.  If it's a list, call GetArray() to get
// a slice of JSONObjects.  To read any given item from the slice, you'll
// need to "Get" that as the right type as well.
// There is one exception: a MAASObject is really a special kind of map,
// so you can read it as either.
// Reading a null item is also an error.  So before you try obj.Get*(),
// first check that obj != nil.
type JSONObject interface {
	// Type of this value:
	// "string", "float64", "map", "maasobject", "array", or "bool".
	Type() string
	// Read as string.
	GetString() (string, error)
	// Read number as float64.
	GetFloat64() (float64, error)
	// Read object as map.
	GetMap() (map[string]JSONObject, error)
	// Read object as MAAS object.
	GetMAASObject() (MAASObject, error)
	// Read list as array.
	GetArray() ([]JSONObject, error)
	// Read as bool.
	GetBool() (bool, error)
}

// Internally, each JSONObject already knows what type it is.  It just
// can't tell the caller yet because the caller may not have the right
// hard-coded variable type.
// So for each JSON type, there is a separate implementation of JSONObject
// that converts only to that type.  Any other conversion is an error.
// One type is special: jsonMAASObject is an object in the MAAS sense.  It
// behaves just like a jsonMap if you want it to, but it also implements
// MAASObject.
type jsonString string
type jsonFloat64 float64
type jsonMap map[string]JSONObject
type jsonArray []JSONObject
type jsonBool bool

// Our JSON processor distinguishes a MAASObject from a jsonMap by the fact
// that it contains a key "resource_uri".  (A regular map might contain the
// same key through sheer coincide, but never mind: you can still treat it
// as a jsonMap and never notice the difference.)
const resourceURI = "resource_uri"

// Internal: turn a completely untyped json.Unmarshal result into a
// JSONObject (with the appropriate implementation of course).
// This function is recursive.  Maps and arrays are deep-copied, with each
// individual value being converted to a JSONObject type.
func maasify(client Client, value interface{}) JSONObject {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case string:
		return jsonString(value.(string))
	case int:
		return jsonFloat64(float64(value.(int)))
	case float64:
		return jsonFloat64(value.(float64))
	case map[string]interface{}:
		original := value.(map[string]interface{})
		result := make(map[string]JSONObject, len(original))
		for key, value := range original {
			result[key] = maasify(client, value)
		}
		if _, ok := result[resourceURI]; ok {
			// If the map contains "resource-uri", we can treat
			// it as a MAAS object.
			return newJSONMAASObject(result, client)
		}
		return jsonMap(result)
	case []interface{}:
		original := value.([]interface{})
		result := make([]JSONObject, len(original))
		for index, value := range original {
			result[index] = maasify(client, value)
		}
		return jsonArray(result)
	case bool:
		return jsonBool(value.(bool))
	}
	msg := fmt.Sprintf("Unknown JSON type, can't be converted to JSONObject: %v", value)
	panic(msg)
}

// Parse a JSON blob into a JSONObject.
func Parse(client Client, input []byte) (JSONObject, error) {
	var obj interface{}
	err := json.Unmarshal(input, &obj)
	if err != nil {
		return nil, err
	}
	return maasify(client, obj), nil
}

// Return error value for failed type conversion.
func failConversion(wanted_type string, obj JSONObject) error {
	msg := fmt.Sprintf("Requested %v, got %v.", wanted_type, obj.Type())
	return errors.New(msg)
}

// Error return values for failure to convert to string.
func failString(obj JSONObject) (string, error) {
	return "", failConversion("string", obj)
}

// Error return values for failure to convert to float64.
func failFloat64(obj JSONObject) (float64, error) {
	return 0.0, failConversion("float64", obj)
}

// Error return values for failure to convert to map.
func failMap(obj JSONObject) (map[string]JSONObject, error) {
	return make(map[string]JSONObject, 0), failConversion("map", obj)
}

// Error return values for failure to convert to MAAS object.
func failMAASObject(obj JSONObject) (MAASObject, error) {
	return jsonMAASObject{}, failConversion("maasobject", obj)
}

// Error return values for failure to convert to array.
func failArray(obj JSONObject) ([]JSONObject, error) {
	return make([]JSONObject, 0), failConversion("array", obj)
}

// Error return values for failure to convert to bool.
func failBool(obj JSONObject) (bool, error) {
	return false, failConversion("bool", obj)
}

// JSONObject implementation for jsonString.
func (jsonString) Type() string                               { return "string" }
func (obj jsonString) GetString() (string, error)             { return string(obj), nil }
func (obj jsonString) GetFloat64() (float64, error)           { return failFloat64(obj) }
func (obj jsonString) GetMap() (map[string]JSONObject, error) { return failMap(obj) }
func (obj jsonString) GetMAASObject() (MAASObject, error)     { return failMAASObject(obj) }
func (obj jsonString) GetArray() ([]JSONObject, error)        { return failArray(obj) }
func (obj jsonString) GetBool() (bool, error)                 { return failBool(obj) }

// JSONObject implementation for jsonFloat64.
func (jsonFloat64) Type() string                               { return "float64" }
func (obj jsonFloat64) GetString() (string, error)             { return failString(obj) }
func (obj jsonFloat64) GetFloat64() (float64, error)           { return float64(obj), nil }
func (obj jsonFloat64) GetMap() (map[string]JSONObject, error) { return failMap(obj) }
func (obj jsonFloat64) GetMAASObject() (MAASObject, error)     { return failMAASObject(obj) }
func (obj jsonFloat64) GetArray() ([]JSONObject, error)        { return failArray(obj) }
func (obj jsonFloat64) GetBool() (bool, error)                 { return failBool(obj) }

// JSONObject implementation for jsonMap.
func (jsonMap) Type() string                     { return "map" }
func (obj jsonMap) GetString() (string, error)   { return failString(obj) }
func (obj jsonMap) GetFloat64() (float64, error) { return failFloat64(obj) }
func (obj jsonMap) GetMap() (map[string]JSONObject, error) {
	return (map[string]JSONObject)(obj), nil
}
func (obj jsonMap) GetMAASObject() (MAASObject, error) { return failMAASObject(obj) }
func (obj jsonMap) GetArray() ([]JSONObject, error)    { return failArray(obj) }
func (obj jsonMap) GetBool() (bool, error)             { return failBool(obj) }

// JSONObject implementation for jsonArray.
func (jsonArray) Type() string                               { return "array" }
func (obj jsonArray) GetString() (string, error)             { return failString(obj) }
func (obj jsonArray) GetFloat64() (float64, error)           { return failFloat64(obj) }
func (obj jsonArray) GetMap() (map[string]JSONObject, error) { return failMap(obj) }
func (obj jsonArray) GetMAASObject() (MAASObject, error)     { return failMAASObject(obj) }
func (obj jsonArray) GetArray() ([]JSONObject, error) {
	return ([]JSONObject)(obj), nil
}
func (obj jsonArray) GetBool() (bool, error) { return failBool(obj) }

// JSONObject implementation for jsonBool.
func (jsonBool) Type() string                               { return "bool" }
func (obj jsonBool) GetString() (string, error)             { return failString(obj) }
func (obj jsonBool) GetFloat64() (float64, error)           { return failFloat64(obj) }
func (obj jsonBool) GetMap() (map[string]JSONObject, error) { return failMap(obj) }
func (obj jsonBool) GetMAASObject() (MAASObject, error)     { return failMAASObject(obj) }
func (obj jsonBool) GetArray() ([]JSONObject, error)        { return failArray(obj) }
func (obj jsonBool) GetBool() (bool, error)                 { return bool(obj), nil }
