// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	. "launchpad.net/gocheck"
	"net/http"
)

func (suite *GomaasapiTestSuite) TestServerListNodesReturnsMAASObject(c *C) {
	URL := "/nodes/?op=list"
	blob := `[{"resource_uri": "/obj1"}, {"resource_uri": "/obj2"}]`
	testServer := newSingleServingServer(URL, blob, http.StatusOK)
	defer testServer.Close()
	client, _ := NewAnonymousClient()
	server := Server{testServer.URL, client}

	result, err := server.ListNodes()

	c.Assert(err, IsNil)
	c.Check(result, Not(IsNil))
	c.Check(len(result), Equals, 2)
	obj1, err := result[0].GetMAASObject()
	c.Assert(err, IsNil)
	c.Check(obj1.URL(), Equals, "/obj1")
}

func (suite *GomaasapiTestSuite) TestServerListNodesReturnsServerError(c *C) {
	URL := "/nodes/?op=list"
	expectedResult := "expected:result"
	testServer := newSingleServingServer(URL, expectedResult, http.StatusBadRequest)
	defer testServer.Close()
	client, _ := NewAnonymousClient()
	server := Server{testServer.URL, client}

	_, err := server.ListNodes()

	c.Assert(err, ErrorMatches, "Error requesting the MAAS server: 400 Bad Request.*")
}
