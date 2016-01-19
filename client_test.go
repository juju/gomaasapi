// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	. "gopkg.in/check.v1"
)

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (*ClientSuite) TestReadAndCloseReturnsEmptyStringForNil(c *C) {
	data, err := readAndClose(nil)
	c.Assert(err, IsNil)
	c.Check(string(data), Equals, "")
}

func (*ClientSuite) TestReadAndCloseReturnsContents(c *C) {
	content := "Stream contents."
	stream := ioutil.NopCloser(strings.NewReader(content))

	data, err := readAndClose(stream)
	c.Assert(err, IsNil)

	c.Check(string(data), Equals, content)
}

func (suite *ClientSuite) TestClientdispatchRequestReturnsServerError(c *C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusBadRequest)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	expectedErrorString := fmt.Sprintf("gomaasapi: got error back from server: 400 Bad Request (%v)", expectedResult)
	c.Check(err.Error(), Equals, expectedErrorString)
	c.Check(err.(ServerError).StatusCode, Equals, 400)
	c.Check(string(result), Equals, expectedResult)
}

func (suite *ClientSuite) TestClientdispatchRequestRetries503(c *C) {
	URI := "/some/url/?param1=test"
	server := newFlakyServer(URI, 503, NumberOfRetries)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)
	content := "content"
	request, err := http.NewRequest("GET", server.URL+URI, ioutil.NopCloser(strings.NewReader(content)))

	_, err = client.dispatchRequest(request)

	c.Check(err, IsNil)
	c.Check(*server.nbRequests, Equals, NumberOfRetries+1)
	expectedRequestsContent := make([][]byte, NumberOfRetries+1)
	for i := 0; i < NumberOfRetries+1; i++ {
		expectedRequestsContent[i] = []byte(content)
	}
	c.Check(*server.requests, DeepEquals, expectedRequestsContent)
}

func (suite *ClientSuite) TestClientdispatchRequestDoesntRetry200(c *C) {
	URI := "/some/url/?param1=test"
	server := newFlakyServer(URI, 200, 10)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)

	request, err := http.NewRequest("GET", server.URL+URI, nil)

	_, err = client.dispatchRequest(request)

	c.Check(err, IsNil)
	c.Check(*server.nbRequests, Equals, 1)
}

func (suite *ClientSuite) TestClientdispatchRequestRetriesIsLimited(c *C) {
	URI := "/some/url/?param1=test"
	// Make the server return 503 responses NumberOfRetries + 1 times.
	server := newFlakyServer(URI, 503, NumberOfRetries+1)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	_, err = client.dispatchRequest(request)

	c.Check(*server.nbRequests, Equals, NumberOfRetries+1)
	c.Check(err.(ServerError).StatusCode, Equals, 503)
}

func (suite *ClientSuite) TestClientDispatchRequestReturnsNonServerError(c *C) {
	client, err := NewAnonymousClient("/foo", "1.0")
	c.Assert(err, IsNil)
	// Create a bad request that will fail to dispatch.
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, IsNil)

	result, err := client.dispatchRequest(request)

	// This type of failure is an error, but not a ServerError.
	c.Check(err, NotNil)
	c.Check(err, Not(FitsTypeOf), ServerError{})
	// For this kind of error, result is guaranteed to be nil.
	c.Check(result, IsNil)
}

func (suite *ClientSuite) TestClientdispatchRequestSignsRequest(c *C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAuthenticatedClient(server.URL, "the:api:key", "1.0")
	c.Assert(err, IsNil)
	request, err := http.NewRequest("GET", server.URL+URI, nil)
	c.Assert(err, IsNil)

	result, err := client.dispatchRequest(request)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	c.Check((*server.requestHeader)["Authorization"][0], Matches, "^OAuth .*")
}

func (suite *ClientSuite) TestClientGetFormatsGetParameters(c *C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, IsNil)
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	fullURI := URI.String() + "?test=123"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)

	result, err := client.Get(URI, "", params)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
}

func (suite *ClientSuite) TestClientGetFormatsOperationAsGetParameter(c *C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, IsNil)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)

	result, err := client.Get(URI, "list", nil)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
}

func (suite *ClientSuite) TestClientPostSendsRequestWithParams(c *C) {
	URI, err := url.Parse("/some/url")
	c.Check(err, IsNil)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Check(err, IsNil)

	result, err := client.Post(URI, "list", params, nil)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	postedValues, err := url.ParseQuery(*server.requestContent)
	c.Check(err, IsNil)
	expectedPostedValues, err := url.ParseQuery("test=123")
	c.Check(err, IsNil)
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
	URI, err := url.Parse("/some/url")
	c.Assert(err, IsNil)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=add"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)
	fileContent := []byte("content")
	files := map[string][]byte{"testfile": fileContent}

	result, err := client.Post(URI, "add", nil, files)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	receivedFileContent, err := extractFileContent(*server.requestContent, server.requestHeader, fullURI, "testfile")
	c.Assert(err, IsNil)
	c.Check(receivedFileContent, DeepEquals, fileContent)
}

func (suite *ClientSuite) TestClientPutSendsRequest(c *C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, IsNil)
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)

	result, err := client.Put(URI, params)

	c.Check(err, IsNil)
	c.Check(string(result), Equals, expectedResult)
	c.Check(*server.requestContent, Equals, "test=123")
}

func (suite *ClientSuite) TestClientDeleteSendsRequest(c *C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, IsNil)
	expectedResult := "expected:result"
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, IsNil)

	err = client.Delete(URI)

	c.Check(err, IsNil)
}

func (suite *ClientSuite) TestNewAnonymousClientEnsuresTrailingSlash(c *C) {
	client, err := NewAnonymousClient("http://example.com/", "1.0")
	c.Check(err, IsNil)
	expectedURL, err := url.Parse("http://example.com/api/1.0/")
	c.Assert(err, IsNil)
	c.Check(client.APIURL, DeepEquals, expectedURL)
}

func (suite *ClientSuite) TestNewAuthenticatedClientEnsuresTrailingSlash(c *C) {
	client, err := NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	c.Check(err, IsNil)
	expectedURL, err := url.Parse("http://example.com/api/1.0/")
	c.Assert(err, IsNil)
	c.Check(client.APIURL, DeepEquals, expectedURL)
}

func (suite *ClientSuite) TestNewAuthenticatedClientParsesApiKey(c *C) {
	// NewAuthenticatedClient returns a plainTextOAuthSigneri configured
	// to use the given API key.
	consumerKey := "consumerKey"
	tokenKey := "tokenKey"
	tokenSecret := "tokenSecret"
	keyElements := []string{consumerKey, tokenKey, tokenSecret}
	apiKey := strings.Join(keyElements, ":")

	client, err := NewAuthenticatedClient("http://example.com/", apiKey, "1.0")

	c.Check(err, IsNil)
	signer := client.Signer.(*plainTextOAuthSigner)
	c.Check(signer.token.ConsumerKey, Equals, consumerKey)
	c.Check(signer.token.TokenKey, Equals, tokenKey)
	c.Check(signer.token.TokenSecret, Equals, tokenSecret)
}

func (suite *ClientSuite) TestNewAuthenticatedClientFailsIfInvalidKey(c *C) {
	client, err := NewAuthenticatedClient("", "invalid-key", "1.0")

	c.Check(err, ErrorMatches, "invalid API key.*")
	c.Check(client, IsNil)

}

func (suite *ClientSuite) TestcomposeAPIURLReturnsURL(c *C) {
	apiurl, err := composeAPIURL("http://example.com/MAAS", "1.0")
	c.Assert(err, IsNil)
	expectedURL, err := url.Parse("http://example.com/MAAS/api/1.0/")
	c.Check(expectedURL, DeepEquals, apiurl)
}
