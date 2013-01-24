// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Affero General Public License version 3 (see the file LICENSE).

package gomaasapi

import (
	. "launchpad.net/gocheck"
	"net/http"
	"net/url"
	"strings"
)

func (suite *GomaasapiTestSuite) TestClientdispatchRequestReturnsError(c *C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	client, _ := NewAnonymousClient()
	server := newSingleServingServer(URI, expectedResult, http.StatusBadRequest)
	defer server.Close()
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	c.Assert(err, ErrorMatches, "Error requesting the MAAS server: 400 Bad Request.*")
	c.Assert(string(result), Equals, expectedResult)
}

func (suite *GomaasapiTestSuite) TestClientdispatchRequestSignsRequest(c *C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	client, _ := NewAuthenticatedClient("the:api:key")
	server := newSingleServingServer(URI, expectedResult, http.StatusOK)
	defer server.Close()
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	c.Assert(err, IsNil)
	c.Assert(string(result), Equals, expectedResult)
	c.Assert((*server.requestHeader)["Authorization"][0], Matches, "^OAuth .*")
}

func (suite *GomaasapiTestSuite) TestClientGetFormatsGetParameters(c *C) {
	URI := "/some/url"
	expectedResult := "expected:result"
	client, _ := NewAnonymousClient()
	params := url.Values{}
	params.Add("op", "list")
	fullURI := URI + "?op=list"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()

	result, err := client.Get(server.URL+URI, params)

	c.Assert(err, IsNil)
	c.Assert(string(result), Equals, expectedResult)
}

func (suite *GomaasapiTestSuite) TestNewAuthenticatedClientParsesApiKey(c *C) {
	// NewAuthenticatedClient returns a pLAINTEXTOAuthSigner configured
	// to use the given API key.
	consumerKey := "consumerKey"
	tokenKey := "tokenKey"
	tokenSecret := "tokenSecret"
	keyElements := []string{consumerKey, tokenKey, tokenSecret}
	apiKey := strings.Join(keyElements, ":")

	client, err := NewAuthenticatedClient(apiKey)

	c.Assert(err, IsNil)
	signer := client.Signer.(pLAINTEXTOAuthSigner)
	c.Assert(signer.token.ConsumerKey, Equals, consumerKey)
	c.Assert(signer.token.TokenKey, Equals, tokenKey)
	c.Assert(signer.token.TokenSecret, Equals, tokenSecret)
}

func (suite *GomaasapiTestSuite) TestNewAuthenticatedClientFailsIfInvalidKey(c *C) {
	client, err := NewAuthenticatedClient("invalid-key")

	c.Assert(err, ErrorMatches, "Invalid API key.*")
	c.Assert(client, IsNil)

}
