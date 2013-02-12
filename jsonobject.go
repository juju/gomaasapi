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
// first check obj.IsNil().
type JSONObject struct {
	// Parsed value.  May actually be any of the types a JSONObject can
	// wrap, except raw bytes.  If the object can only be interpreted
	// as raw bytes, this will be nil.
	value interface{}
	// Raw bytes, if this object was parsed directly from an API response.
	// Is nil for sub-objects found within other objects.  An object that
	// was parsed directly from a response can be both raw bytes and some
	// other value at the same  time.
	// For example, "[]" looks like a JSON list, so you can read it as an
	// array.  But it may also be the raw contents of a file that just
	// happens to look like JSON, and so you can read it as raw bytes as
	// well.
	bytes []byte
	// Client for further communication with the API.
	client Client
}

// Our JSON processor distinguishes a MAASObject from a jsonMap by the fact
// that it contains a key "resource_uri".  (A regular map might contain the
// same key through sheer coincide, but never mind: you can still treat it
// as a jsonMap and never notice the difference.)
const resourceURI = "resource_uri"

// maasify turns a completely untyped json.Unmarshal result into a JSONObject
// (with the appropriate implementation of course).  This function is
// recursive.  Maps and arrays are deep-copied, with each individual value
// being converted to a JSONObject type.
func maasify(client Client, value interface{}) JSONObject {
	if value == nil {
		return JSONObject{}
	}
	switch value.(type) {
	case string, float64, bool:
		return JSONObject{value: value}
	case map[string]interface{}:
		original := value.(map[string]interface{})
		result := make(map[string]JSONObject, len(original))
		for key, value := range original {
			result[key] = maasify(client, value)
		}
		return JSONObject{value: result, client: client}
	case []interface{}:
		original := value.([]interface{})
		result := make([]JSONObject, len(original))
		for index, value := range original {
			result[index] = maasify(client, value)
		}
		return JSONObject{value: result}
	}
	msg := fmt.Sprintf("Unknown JSON type, can't be converted to JSONObject: %v", value)
	panic(msg)
}

// Parse a JSON blob into a JSONObject.
func Parse(client Client, input []byte) (JSONObject, error) {
	var obj interface{}
	err := json.Unmarshal(input, &obj)
	if err != nil {
		return JSONObject{}, err
	}
	return maasify(client, obj), nil
}

// Return error value for failed type conversion.
func failConversion(wantedType string, obj JSONObject) error {
	msg := fmt.Sprintf("Requested %v, got %T.", wantedType, obj.value)
	return errors.New(msg)
}

// IsNil tells you whether a JSONObject is a JSON "null."
// There is one irregularity.  If the original JSON blob was actually raw
// data, not JSON, then its IsNil will return false because the object
// contains the binary data as a non-nil value.  But, if the original JSON
// blob consisted of a null, then IsNil returns true even though you can
// still retrieve binary data from it.
func (obj JSONObject) IsNil() bool {
	return obj.value == nil
}

func (obj JSONObject) GetString() (value string, err error) {
	value, ok := obj.value.(string)
	if !ok {
		err = failConversion("string", obj)
	}
	return
}

func (obj JSONObject) GetFloat64() (value float64, err error) {
	value, ok := obj.value.(float64)
	if !ok {
		err = failConversion("float64", obj)
	}
	return
}

func (obj JSONObject) GetMap() (value map[string]JSONObject, err error) {
	value, ok := obj.value.(map[string]JSONObject)
	if !ok {
		err = failConversion("map", obj)
	}
	return
}

func (obj JSONObject) GetArray() (value []JSONObject, err error) {
	value, ok := obj.value.([]JSONObject)
	if !ok {
		err = failConversion("array", obj)
	}
	return
}

func (obj JSONObject) GetBool() (value bool, err error) {
	value, ok := obj.value.(bool)
	if !ok {
		err = failConversion("bool", obj)
	}
	return
}
