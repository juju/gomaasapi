// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
)

type Server struct {
	URL    string
	client *Client
}

func (server *Server) listNodes() []Node {
	// Do something like (warning, completely untested code):
	listURL := server.URL + "nodes/"
	result, _ := (*server.client).Get(listURL, nil)
	var nodeList []Node
	_ = json.Unmarshal(result, &nodeList)
	return nodeList
}
