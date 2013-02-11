// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

// NewMAAS returns an interface to the MAAS API as a MAASObject.
func NewMAAS(client Client) MAASObject {
	attrs := map[string]interface{}{resourceURI: client.BaseURL.String()}
	return newJSONMAASObject(attrs, client)
}
