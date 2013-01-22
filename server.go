// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

type Server struct {
	URL    string
	client *Client
}

func (server *Server) listNodes() []Node {
	panic("Not implemented yet")
}
