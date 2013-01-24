// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Affero General Public License version 3 (see the file LICENSE).

package gomaasapi

import (
	. "launchpad.net/gocheck"
	"net/http"
)

func (suite *GomaasapiTestSuite) TestServerListNodesReturnsMAASObject(c *C) {
	URL := "/nodes/?op=list"
	expectedResult := "expected:result"
	testServer := newSingleServingServer(URL, expectedResult, http.StatusOK)
	defer testServer.Close()
	client, _ := NewAnonymousClient()
	server := Server{testServer.URL, client}

	result, err := server.ListNodes()

	c.Assert(err, IsNil)
	// TODO: Really test result.
	c.Assert(result, Not(IsNil))
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
