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

	// Utility method to extract a string field from this MAAS object.
	GetField(name string) (string, error)
	// URL for this MAAS object.
	URL() *url.URL
	// Resource URI for this MAAS object.
	URI() *url.URL
	// Retrieve the MAAS object located at thisObject.URI()+name.
	GetSubObject(name string) MAASObject
	// Retrieve this MAAS object.
	Get() (MAASObject, error)
	// Write this MAAS object.
	Post(params url.Values) (JSONObject, error)
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
	uri    *url.URL
}

// newJSONMAASObject creates a new MAAS object.  It will panic if the given map
// does not contain a valid URL for the 'resource_uri' key.
func newJSONMAASObject(jmap jsonMap, client Client) jsonMAASObject {
	const panicPrefix = "Error processing MAAS object: "
	uriObj, ok := jmap[resourceURI]
	if !ok {
		panic(errors.New(panicPrefix + "no 'resource_uri' key present in the given jsonMap."))
	}
	uriString, err := uriObj.GetString()
	if err != nil {
		panic(errors.New(panicPrefix + "the value of 'resource_uri' is not a string."))
	}
	uri, err := url.Parse(uriString)
	if err != nil {
		panic(errors.New(panicPrefix + "the value of 'resource_uri' is not a valid URL."))
	}
	return jsonMAASObject{jmap, client, uri}
}

var _ JSONObject = (*jsonMAASObject)(nil)
var _ MAASObject = (*jsonMAASObject)(nil)

// JSONObject implementation for jsonMAASObject.
func (jsonMAASObject) Type() string                               { return "maasobject" }
func (obj jsonMAASObject) GetString() (string, error)             { return failString(obj) }
func (obj jsonMAASObject) GetFloat64() (float64, error)           { return failFloat64(obj) }
func (obj jsonMAASObject) GetMap() (map[string]JSONObject, error) { return obj.jsonMap.GetMap() }
func (obj jsonMAASObject) GetMAASObject() (MAASObject, error)     { return obj, nil }
func (obj jsonMAASObject) GetArray() ([]JSONObject, error)        { return failArray(obj) }
func (obj jsonMAASObject) GetBool() (bool, error)                 { return failBool(obj) }

// MAASObject implementation for jsonMAASObject.

func (obj jsonMAASObject) GetField(name string) (string, error) {
	return obj.jsonMap[name].GetString()
}

func (obj jsonMAASObject) URI() *url.URL {
	// Duplicate the URL.
	uri, err := url.Parse(obj.uri.String())
	if err != nil {
		panic(err)
	}
	return uri
}

func (obj jsonMAASObject) URL() *url.URL {
	return obj.client.GetURL(obj.URI())
}

func (obj jsonMAASObject) GetSubObject(name string) MAASObject {
	uri := obj.URI()
	uri.Path = EnsureTrailingSlash(JoinURLs(uri.Path, name))
	input := map[string]JSONObject{resourceURI: jsonString(uri.String())}
	return newJSONMAASObject(jsonMap(input), obj.client)
}

var NotImplemented = errors.New("Not implemented")

func (obj jsonMAASObject) Get() (MAASObject, error) {
	uri := obj.URI()
	result, err := obj.client.Get(uri, "", url.Values{})
	if err != nil {
		return nil, err
	}
	jsonObj, err := Parse(obj.client, result)
	if err != nil {
		return nil, err
	}
	return jsonObj.GetMAASObject()
}

func (obj jsonMAASObject) Post(params url.Values) (JSONObject, error) {
	uri := obj.URI()
	result, err := obj.client.Post(uri, "", params)
	if err != nil {
		return nil, err
	}
	return Parse(obj.client, result)
}

func (obj jsonMAASObject) Update(params url.Values) (MAASObject, error) {
	uri := obj.URI()
	result, err := obj.client.Put(uri, params)
	if err != nil {
		return nil, err
	}
	jsonObj, err := Parse(obj.client, result)
	if err != nil {
		return nil, err
	}
	return jsonObj.GetMAASObject()
}

func (obj jsonMAASObject) Delete() error {
	uri := obj.URI()
	return obj.client.Delete(uri)
}

func (obj jsonMAASObject) CallGet(operation string, params url.Values) (JSONObject, error) {
	uri := obj.URI()
	result, err := obj.client.Get(uri, operation, params)
	if err != nil {
		return nil, err
	}
	return Parse(obj.client, result)
}

func (obj jsonMAASObject) CallPost(operation string, params url.Values) (JSONObject, error) {
	uri := obj.URI()
	result, err := obj.client.Post(uri, operation, params)
	if err != nil {
		return nil, err
	}
	return Parse(obj.client, result)
}
