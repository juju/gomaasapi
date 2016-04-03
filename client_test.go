// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type ClientSuite struct{}

var _ = gc.Suite(&ClientSuite{})

func (*ClientSuite) TestReadAndCloseReturnsEmptyStringForNil(c *gc.C) {
	data, err := readAndClose(nil)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(data), gc.Equals, "")
}

func (*ClientSuite) TestReadAndCloseReturnsContents(c *gc.C) {
	content := "Stream contents."
	stream := ioutil.NopCloser(strings.NewReader(content))

	data, err := readAndClose(stream)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(string(data), gc.Equals, content)
}

func (suite *ClientSuite) TestClientdispatchRequestReturnsServerError(c *gc.C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusBadRequest)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	expectedErrorString := fmt.Sprintf("ServerError: 400 Bad Request (%v)", expectedResult)
	c.Check(err.Error(), gc.Equals, expectedErrorString)

	svrError, ok := GetServerError(err)
	c.Assert(ok, jc.IsTrue)
	c.Check(svrError.StatusCode, gc.Equals, 400)
	c.Check(string(result), gc.Equals, expectedResult)
}

func (suite *ClientSuite) TestClientdispatchRequestRetries503(c *gc.C) {
	URI := "/some/url/?param1=test"
	server := newFlakyServer(URI, 503, NumberOfRetries)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)
	content := "content"
	request, err := http.NewRequest("GET", server.URL+URI, ioutil.NopCloser(strings.NewReader(content)))

	_, err = client.dispatchRequest(request)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(*server.nbRequests, gc.Equals, NumberOfRetries+1)
	expectedRequestsContent := make([][]byte, NumberOfRetries+1)
	for i := 0; i < NumberOfRetries+1; i++ {
		expectedRequestsContent[i] = []byte(content)
	}
	c.Check(*server.requests, jc.DeepEquals, expectedRequestsContent)
}

func (suite *ClientSuite) TestClientdispatchRequestDoesntRetry200(c *gc.C) {
	URI := "/some/url/?param1=test"
	server := newFlakyServer(URI, 200, 10)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)

	request, err := http.NewRequest("GET", server.URL+URI, nil)

	_, err = client.dispatchRequest(request)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(*server.nbRequests, gc.Equals, 1)
}

func (suite *ClientSuite) TestClientdispatchRequestRetriesIsLimited(c *gc.C) {
	URI := "/some/url/?param1=test"
	// Make the server return 503 responses NumberOfRetries + 1 times.
	server := newFlakyServer(URI, 503, NumberOfRetries+1)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	_, err = client.dispatchRequest(request)

	c.Check(*server.nbRequests, gc.Equals, NumberOfRetries+1)
	svrError, ok := GetServerError(err)
	c.Assert(ok, jc.IsTrue)
	c.Assert(svrError.StatusCode, gc.Equals, 503)
}

func (suite *ClientSuite) TestClientDispatchRequestReturnsNonServerError(c *gc.C) {
	client, err := NewAnonymousClient("/foo", "1.0")
	c.Assert(err, jc.ErrorIsNil)
	// Create a bad request that will fail to dispatch.
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, jc.ErrorIsNil)

	result, err := client.dispatchRequest(request)
	c.Check(err, gc.NotNil)
	// This type of failure is an error, but not a ServerError.
	_, ok := GetServerError(err)
	c.Assert(ok, jc.IsFalse)
	// For this kind of error, result is guaranteed to be nil.
	c.Check(result, gc.IsNil)
}

func (suite *ClientSuite) TestClientdispatchRequestSignsRequest(c *gc.C) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAuthenticatedClient(server.URL, "the:api:key", "1.0")
	c.Assert(err, jc.ErrorIsNil)
	request, err := http.NewRequest("GET", server.URL+URI, nil)
	c.Assert(err, jc.ErrorIsNil)

	result, err := client.dispatchRequest(request)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(result), gc.Equals, expectedResult)
	c.Check((*server.requestHeader)["Authorization"][0], gc.Matches, "^OAuth .*")
}

func (suite *ClientSuite) TestClientGetFormatsGetParameters(c *gc.C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, jc.ErrorIsNil)
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	fullURI := URI.String() + "?test=123"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)

	result, err := client.Get(URI, "", params)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(result), gc.Equals, expectedResult)
}

func (suite *ClientSuite) TestClientGetFormatsOperationAsGetParameter(c *gc.C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, jc.ErrorIsNil)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)

	result, err := client.Get(URI, "list", nil)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(result), gc.Equals, expectedResult)
}

func (suite *ClientSuite) TestClientPostSendsRequestWithParams(c *gc.C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, jc.ErrorIsNil)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)

	result, err := client.Post(URI, "list", params, nil)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(result), gc.Equals, expectedResult)
	postedValues, err := url.ParseQuery(*server.requestContent)
	c.Assert(err, jc.ErrorIsNil)
	expectedPostedValues, err := url.ParseQuery("test=123")
	c.Assert(err, jc.ErrorIsNil)
	c.Check(postedValues, jc.DeepEquals, expectedPostedValues)
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

func (suite *ClientSuite) TestClientPostSendsMultipartRequest(c *gc.C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, jc.ErrorIsNil)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=add"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)
	fileContent := []byte("content")
	files := map[string][]byte{"testfile": fileContent}

	result, err := client.Post(URI, "add", nil, files)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(result), gc.Equals, expectedResult)
	receivedFileContent, err := extractFileContent(*server.requestContent, server.requestHeader, fullURI, "testfile")
	c.Assert(err, jc.ErrorIsNil)
	c.Check(receivedFileContent, jc.DeepEquals, fileContent)
}

func (suite *ClientSuite) TestClientPutSendsRequest(c *gc.C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, jc.ErrorIsNil)
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)

	result, err := client.Put(URI, params)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(string(result), gc.Equals, expectedResult)
	c.Check(*server.requestContent, gc.Equals, "test=123")
}

func (suite *ClientSuite) TestClientDeleteSendsRequest(c *gc.C) {
	URI, err := url.Parse("/some/url")
	c.Assert(err, jc.ErrorIsNil)
	expectedResult := "expected:result"
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	c.Assert(err, jc.ErrorIsNil)

	err = client.Delete(URI)

	c.Assert(err, jc.ErrorIsNil)
}

func (suite *ClientSuite) TestNewAnonymousClientEnsuresTrailingSlash(c *gc.C) {
	client, err := NewAnonymousClient("http://example.com/", "1.0")
	c.Assert(err, jc.ErrorIsNil)
	expectedURL, err := url.Parse("http://example.com/api/1.0/")
	c.Assert(err, jc.ErrorIsNil)
	c.Check(client.APIURL, jc.DeepEquals, expectedURL)
}

func (suite *ClientSuite) TestNewAuthenticatedClientEnsuresTrailingSlash(c *gc.C) {
	client, err := NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	c.Assert(err, jc.ErrorIsNil)
	expectedURL, err := url.Parse("http://example.com/api/1.0/")
	c.Assert(err, jc.ErrorIsNil)
	c.Check(client.APIURL, jc.DeepEquals, expectedURL)
}

func (suite *ClientSuite) TestNewAuthenticatedClientParsesApiKey(c *gc.C) {
	// NewAuthenticatedClient returns a plainTextOAuthSigneri configured
	// to use the given API key.
	consumerKey := "consumerKey"
	tokenKey := "tokenKey"
	tokenSecret := "tokenSecret"
	keyElements := []string{consumerKey, tokenKey, tokenSecret}
	apiKey := strings.Join(keyElements, ":")

	client, err := NewAuthenticatedClient("http://example.com/", apiKey, "1.0")

	c.Assert(err, jc.ErrorIsNil)
	signer := client.Signer.(*plainTextOAuthSigner)
	c.Check(signer.token.ConsumerKey, gc.Equals, consumerKey)
	c.Check(signer.token.TokenKey, gc.Equals, tokenKey)
	c.Check(signer.token.TokenSecret, gc.Equals, tokenSecret)
}

func (suite *ClientSuite) TestNewAuthenticatedClientFailsIfInvalidKey(c *gc.C) {
	client, err := NewAuthenticatedClient("", "invalid-key", "1.0")

	c.Check(err, gc.ErrorMatches, "invalid API key.*")
	c.Check(client, gc.IsNil)

}

func (suite *ClientSuite) TestcomposeAPIURLReturnsURL(c *gc.C) {
	apiurl, err := composeAPIURL("http://example.com/MAAS", "1.0")
	c.Assert(err, jc.ErrorIsNil)
	expectedURL, err := url.Parse("http://example.com/MAAS/api/1.0/")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(expectedURL, jc.DeepEquals, apiurl)
}
