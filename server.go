// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"net/url"
)

type Server struct {
	URL    string
	Client *Client
}

func (server *Server) ListNodes() ([]JSONObject, error) {
	listURL := server.URL + "/nodes/"
	params := url.Values{}
	params.Add("op", "list")
	result, err := server.Client.Get(listURL, params)
	if err != nil {
		return nil, err
	}
	jsonobj, err := Parse(*server.Client, result)
	if err != nil {
		return nil, err
	}
	return jsonobj.GetArray()
}
