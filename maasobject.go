// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"errors"
	"net/url"
)

// MAASObject represents a MAAS object as returned by the MAAS API, such as a
// Node or a Tag.
// This is a special kind of JSONObject.  A MAAS API call will usually return
// either a MAASObject or a list of MAASObjects.  (The list itself will be
// wrapped in a JSONObject).
type MAASObject interface {
	JSONObject

	// Resource URI for this MAAS object.
	URL() string
	// Retrieve this MAAS object.
	Get() (MAASObject, error)
	// Write this MAAS object.
	Post(params url.Values) (MAASObject, error)
	// Update this MAAS object with the given values.
	Update(params url.Values) (MAASObject, error)
	// Delete this MAAS object.
	Delete() error
	// Invoke a GET-based method on this MAAS object.
	CallGet(operation string, params url.Values) (JSONObject, error)
	// Invoke a POST-based method on this MAAS object.
	CallPost(operation string, params url.Values) (JSONObject, error)
}

// JSONObject implementation for a MAAS object.  From a decoding perspective,
// a jsonMAASObject is just like a jsonMap except it contains a key
// "resource_uri", and it keeps track of the Client you got it from so that
// you can invoke API methods directly on their MAAS objects.
// jsonMAASObject implements both JSONObject and MAASObject.
type jsonMAASObject struct {
	jsonMap
	client Client
}

// JSONObject implementation for jsonMAASObject.
func (jsonMAASObject) Type() string                               { return "maasobject" }
func (obj jsonMAASObject) GetString() (string, error)             { return failString(obj) }
func (obj jsonMAASObject) GetFloat64() (float64, error)           { return failFloat64(obj) }
func (obj jsonMAASObject) GetMap() (map[string]JSONObject, error) { return obj.jsonMap.GetMap() }
func (obj jsonMAASObject) GetMAASObject() (MAASObject, error)           { return obj, nil }
func (obj jsonMAASObject) GetArray() ([]JSONObject, error)        { return failArray(obj) }
func (obj jsonMAASObject) GetBool() (bool, error)                 { return failBool(obj) }

// MAASObject implementation for jsonMAASObject.

func (obj jsonMAASObject) URL() string {
	contents, err := obj.GetMap()
	if err != nil {
		panic("Unexpected failure converting jsonMAASObject to maasMap.")
	}
	url, err := contents[resource_uri].GetString()
	if err != nil {
		panic("Unexpected failure reading jsonMAASObject's URL.")
	}
	return url
}

var NotImplemented = errors.New("Not implemented")

func (obj jsonMAASObject) Get() (MAASObject, error) {
	return jsonMAASObject{}, NotImplemented
}

func (obj jsonMAASObject) Post(params url.Values) (MAASObject, error) {
	return jsonMAASObject{}, NotImplemented
}

func (obj jsonMAASObject) Update(params url.Values) (MAASObject, error) {
	return jsonMAASObject{}, NotImplemented
}

func (obj jsonMAASObject) Delete() error { return NotImplemented }

func (obj jsonMAASObject) CallGet(operation string, params url.Values) (JSONObject, error) {
	return nil, NotImplemented
}

func (obj jsonMAASObject) CallPost(operation string, params url.Values) (JSONObject, error) {
	return nil, NotImplemented
}
