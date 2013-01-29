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
	URL() (*url.URL, error)
	// Resource URI for this MAAS object.
	URI() (*url.URL, error)
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

func (obj jsonMAASObject) URI() (*url.URL, error) {
	contents, err := obj.GetMap()
	if err != nil {
		panic("Unexpected failure converting jsonMAASObject to maasMap.")
	}
	urlString, err := contents[resource_uri].GetString()
	if err != nil {
		return &url.URL{}, err
	}
	return url.Parse(urlString)
}

func (obj jsonMAASObject) URL() (*url.URL, error) {
	uri, err := obj.URI()
	if err != nil {
		return &url.URL{}, err
	}
	return obj.client.GetURL(uri), nil
}

func (obj jsonMAASObject) GetSubObject(name string) MAASObject {
	uri, err := obj.URI()
	if err != nil {
		panic("Unexpected failure reading jsonMAASObject's URL.")
	}
	uri.Path = EnsureTrailingSlash(JoinURLs(uri.Path, name))
	input := map[string]JSONObject{resource_uri: jsonString(uri.String())}
	return jsonMAASObject{jsonMap: jsonMap(input), client: obj.client}
}

var NotImplemented = errors.New("Not implemented")

func (obj jsonMAASObject) Get() (MAASObject, error) {
	uri, err := obj.URI()
	if err != nil {
		return nil, err
	}
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
	uri, err := obj.URI()
	if err != nil {
		return nil, err
	}
	result, err := obj.client.Post(uri, "", params)
	if err != nil {
		return nil, err
	}
	return Parse(obj.client, result)
}

func (obj jsonMAASObject) Update(params url.Values) (MAASObject, error) {
	uri, err := obj.URI()
	if err != nil {
		return nil, err
	}
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
	uri, err := obj.URI()
	if err != nil {
		return err
	}
	return obj.client.Delete(uri)
}

func (obj jsonMAASObject) CallGet(operation string, params url.Values) (JSONObject, error) {
	uri, err := obj.URI()
	if err != nil {
		return nil, err
	}
	result, err := obj.client.Get(uri, operation, params)
	if err != nil {
		return nil, err
	}
	return Parse(obj.client, result)
}

func (obj jsonMAASObject) CallPost(operation string, params url.Values) (JSONObject, error) {
	uri, err := obj.URI()
	if err != nil {
		return nil, err
	}
	result, err := obj.client.Post(uri, operation, params)
	if err != nil {
		return nil, err
	}
	return Parse(obj.client, result)
}
