// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"bytes"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
	"net/url"
	"strings"
)

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (suite *ClientSuite) TestClientdispatchRequestReturnsError(c *C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusBadRequest)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	c.Check(err, ErrorMatches, "gomaasapi: got error back from server: 400 Bad Request.*")
	c.Check(string(result), Equals, expectedResult)
}

func (suite *ClientSuite) TestClientdispatchRequestSignsRequest(c *C) {
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

func (suite *ClientSuite) TestClientGetFormatsGetParameters(c *C) {
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

func (suite *ClientSuite) TestClientGetFormatsOperationAsGetParameter(c *C) {
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

func (suite *ClientSuite) TestClientPostSendsRequestWithParams(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)

	result, err := client.Post(URI, "list", params, nil)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	postedValues, err := url.ParseQuery(*server.requestContent)
	c.Check(err, IsNil)
	expectedPostedValues, _ := url.ParseQuery("test=123")
	c.Check(postedValues, DeepEquals, expectedPostedValues)
}

// extractFileContent extracts from the request built using 'requestContent',
// 'requestHeader' and 'requestURL', the file named 'filename'.
func extractFileContent(requestContent string, requestHeader *http.Header, requestURL string, filename string) ([]byte, error) {
	// Recreate the request from server.requestContent to use the parsing
	// utility from the http package (http.Request.FormFile).
	request, err := http.NewRequest("POST", requestURL, bytes.NewBufferString(requestContent))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", requestHeader.Get("Content-Type"))
	file, _, err := request.FormFile("testfile")
	if err != nil {
		return nil, err
	}
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

func (suite *ClientSuite) TestClientPostSendsMultipartRequest(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=add"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)
	fileContent := []byte("content")
	files := map[string][]byte{"testfile": fileContent}

	result, err := client.Post(URI, "add", url.Values{}, files)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	receivedFileContent, err := extractFileContent(*server.requestContent, server.requestHeader, fullURI, "testfile")
	c.Check(err, IsNil)
	c.Check(receivedFileContent, DeepEquals, fileContent)
}

func (suite *ClientSuite) TestClientPutSendsRequest(c *C) {
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

func (suite *ClientSuite) TestClientDeleteSendsRequest(c *C) {
	URI, _ := url.Parse("/some/url")
	expectedResult := "expected:result"
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, _ := NewAnonymousClient(server.URL)

	err := client.Delete(URI)

	c.Check(err, IsNil)
}

func (suite *ClientSuite) TestNewAuthenticatedClientParsesApiKey(c *C) {
	// NewAuthenticatedClient returns a plainTextOAuthSigneri configured
	// to use the given API key.
	consumerKey := "consumerKey"
	tokenKey := "tokenKey"
	tokenSecret := "tokenSecret"
	keyElements := []string{consumerKey, tokenKey, tokenSecret}
	apiKey := strings.Join(keyElements, ":")

	client, err := NewAuthenticatedClient("http://example.com/api", apiKey)

	c.Check(err, IsNil)
	signer := client.Signer.(*plainTextOAuthSigner)
	c.Check(signer.token.ConsumerKey, Equals, consumerKey)
	c.Check(signer.token.TokenKey, Equals, tokenKey)
	c.Check(signer.token.TokenSecret, Equals, tokenSecret)
}

func (suite *ClientSuite) TestNewAuthenticatedClientFailsIfInvalidKey(c *C) {
	client, err := NewAuthenticatedClient("", "invalid-key")

	c.Check(err, ErrorMatches, "invalid API key.*")
	c.Check(client, IsNil)

}
