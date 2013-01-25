// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"errors"
	"net/url"
)


// MAASModel represents a model object as returned by the MAAS API.  This is
// a special kind of JSONObject.  A MAAS API call will usually return either
// a MAASModel or a list of MAASModels.  (The list itself will be wrapped in
// a JSONObject).
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
	CallGet(operation string, params url.Values) (JSONObject, error)
	// Invoke a POST-based method on this model object.
	CallPost(operation string, params url.Values) (JSONObject, error)
}

// JSONObject implementation for a MAAS model object.
// Implements both JSONObject and MAASModel.
type maasModel jsonMap


// JSONObject implementation for maasModel.
func (maasModel) Type() string { return "model" }
func (obj maasModel) GetString() (string, error) { return failString(obj) }
func (obj maasModel) GetFloat64() (float64, error) { return failFloat64(obj) }
func (obj maasModel) GetMap() (map[string]JSONObject, error) {
	return (map[string]JSONObject)(obj), nil
}
func (obj maasModel) GetModel() (MAASModel, error) {
	return maasModel(obj), nil
}
func (obj maasModel) GetArray() ([]JSONObject, error) { return failArray(obj) }
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

func (obj maasModel) CallGet(operation string, params url.Values) (JSONObject, error) {
	return nil, NotImplemented
}

func (obj maasModel) CallPost(operation string, params url.Values) (JSONObject, error) {
	return nil, NotImplemented
}
