// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

// MAASObject is a wrapper around a JSON structure which provides
// methods to extract data from that structure.
type MAASObject struct {
}

func NewMAASObject(json []byte) (*MAASObject, error) {
	// Not implemented.
	return &MAASObject{}, nil
}

func NewMAASObjectList(json []byte) ([]*MAASObject, error) {
	// Not implemented.
	list := []*MAASObject{&MAASObject{}}
	return list, nil
}
