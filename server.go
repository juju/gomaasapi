// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"log"
)

type Server struct {
	URL    string
	client *Client
}

func (server *Server) listNodes() ([]*MAASObject, error) {
	// Do something like (warning, completely untested code):
	listURL := server.URL + "nodes/"
	result, err := (*server.client).Get(listURL, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	list, errJson := NewMAASObjectList(result)
	if errJson != nil {
		log.Println(errJson)
		return nil, errJson
	}
	return list, nil
}
