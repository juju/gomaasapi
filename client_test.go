// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

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
	server := newSingleServingServer(URI, expectedResult, http.StatusBadRequest)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	c.Check(err, ErrorMatches, "Error requesting the MAAS server: 400 Bad Request.*")
	c.Check(string(result), Equals, expectedResult)
}

func (suite *GomaasapiTestSuite) TestClientdispatchRequestSignsRequest(c *C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAuthenticatedClient(server.URL, "the:api:key")
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	c.Check((*server.requestHeader)["Authorization"][0], Matches, "^OAuth .*")
}

func (suite *GomaasapiTestSuite) TestClientGetFormatsGetParameters(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	fullURI := URI.String() + "?test=123"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)

	result, err := client.Get(URI, "", params)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
}

func (suite *GomaasapiTestSuite) TestClientGetFormatsOperationAsGetParameter(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)

	result, err := client.Get(URI, "list", url.Values{})

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
}

func (suite *GomaasapiTestSuite) TestClientPostSendsRequest(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)

	result, err := client.Post(URI, "list", params)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	c.Check(*server.requestContent, Equals, "test=123")
}

func (suite *GomaasapiTestSuite) TestClientPutSendsRequest(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)

	result, err := client.Put(URI, params)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	c.Check(*server.requestContent, Equals, "test=123")
}

func (suite *GomaasapiTestSuite) TestClientDeleteSendsRequest(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)

	err := client.Delete(URI)

	c.Check(err, IsNil)
}

func (suite *GomaasapiTestSuite) TestNewAuthenticatedClientParsesApiKey(c *C) {
	// NewAuthenticatedClient returns a _PLAINTEXTOAuthSigner configured
	// to use the given API key.
	consumerKey := "consumerKey"
	tokenKey := "tokenKey"
	tokenSecret := "tokenSecret"
	keyElements := []string{consumerKey, tokenKey, tokenSecret}
	apiKey := strings.Join(keyElements, ":")

	client, err := NewAuthenticatedClient("http://example.com/api", apiKey)

	c.Check(err, IsNil)
	signer := client.Signer.(_PLAINTEXTOAuthSigner)
	c.Check(signer.token.ConsumerKey, Equals, consumerKey)
	c.Check(signer.token.TokenKey, Equals, tokenKey)
	c.Check(signer.token.TokenSecret, Equals, tokenSecret)
}

func (suite *GomaasapiTestSuite) TestNewAuthenticatedClientFailsIfInvalidKey(c *C) {
	client, err := NewAuthenticatedClient("", "invalid-key")

	c.Check(err, ErrorMatches, "Invalid API key.*")
	c.Check(client, IsNil)

}
