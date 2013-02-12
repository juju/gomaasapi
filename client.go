// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// Client represents a way ot communicating with a MAAS API instance.
// It is stateless, so it can have concurrent requests in progress.
type Client struct {
	BaseURL *url.URL
	Signer  OAuthSigner
}

// dispatchRequest sends a request to the server, and interprets the response.
func (client Client) dispatchRequest(request *http.Request) ([]byte, error) {
	client.Signer.OAuthSign(request)
	httpClient := http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return body, fmt.Errorf("gomaasapi: got error back from server: %v", response.Status)
	}
	return body, nil
}

// GetURL returns the URL to a given resource on the API, based on its URI.
// The resource URI may be absolute or relative; either way the result is a
// full absolute URL including the network part.
func (client Client) GetURL(uri *url.URL) *url.URL {
	return client.BaseURL.ResolveReference(uri)
}

// Get performs an HTTP "GET" to the API.  This may be either an API method
// invocation (if you pass its name in "operation") or plain resource
// retrieval (if you leave "operation" blank).
func (client Client) Get(uri *url.URL, operation string, parameters url.Values) ([]byte, error) {
	opParameter := parameters.Get("op")
	if opParameter != "" {
		errString := fmt.Sprintf("The parameters contain a value for '%s' which is reserved parameter.")
		return nil, errors.New(errString)
	}
	if operation != "" {
		parameters.Set("op", operation)
	}
	queryUrl := client.GetURL(uri)
	queryUrl.RawQuery = parameters.Encode()
	request, err := http.NewRequest("GET", queryUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	return client.dispatchRequest(request)
}

// writeMultiPartFiles writes the given files as parts of a multipart message
// using the given writer.
func writeMultiPartFiles(writer *multipart.Writer, files map[string][]byte) {
	for fileName, fileContent := range files {

		fw, err := writer.CreateFormFile(fileName, fileName)
		if err != nil {
			panic(err)
		}
		io.Copy(fw, bytes.NewBuffer(fileContent))
	}
}

// writeMultiPartParams writes the given parameters as parts of a multipart
// message using the given writer.
func writeMultiPartParams(writer *multipart.Writer, parameters url.Values) {
	for key, values := range parameters {
		for _, value := range values {
			fw, err := writer.CreateFormField(key)
			if err != nil {
				panic(err)
			}
			buffer := bytes.NewBufferString(value)
			io.Copy(fw, buffer)
		}
	}

}

// nonIdempotentRequest implements the common functionality of PUT and POST
// requests (but not GET or DELETE requests).
func (client Client) nonIdempotentRequest(method string, uri *url.URL, parameters url.Values, files map[string][]byte) ([]byte, error) {
	var request *http.Request
	var err error
	if files != nil {
		// files is not nil, create a multipart request.
		buf := new(bytes.Buffer)
		writer := multipart.NewWriter(buf)
		writeMultiPartFiles(writer, files)
		writeMultiPartParams(writer, parameters)
		writer.Close()
		url := client.GetURL(uri)
		request, err = http.NewRequest(method, url.String(), buf)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", writer.FormDataContentType())
	} else {
		url := client.GetURL(uri)
		request, err = http.NewRequest(method, url.String(), strings.NewReader(string(parameters.Encode())))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return client.dispatchRequest(request)
}

// Post performs an HTTP "POST" to the API.  This may be either an API method
// invocation (if you pass its name in "operation") or plain resource
// retrieval (if you leave "operation" blank).
func (client Client) Post(uri *url.URL, operation string, parameters url.Values, files map[string][]byte) ([]byte, error) {
	parameters.Set("op", operation)
	return client.nonIdempotentRequest("POST", uri, parameters, files)
}

// Put updates an object on the API, using an HTTP "PUT" request.
func (client Client) Put(uri *url.URL, parameters url.Values) ([]byte, error) {
	return client.nonIdempotentRequest("PUT", uri, parameters, nil)
}

// Delete deletes an object on the API, using an HTTP "DELETE" request.
func (client Client) Delete(uri *url.URL) error {
	url := client.GetURL(uri)
	request, err := http.NewRequest("DELETE", url.String(), strings.NewReader(""))
	if err != nil {
		return err
	}
	_, err = client.dispatchRequest(request)
	if err != nil {
		return err
	}
	return nil
}

// Anonymous "signature method" implementation.
type anonSigner struct{}

func (signer anonSigner) OAuthSign(request *http.Request) error {
	return nil
}

// *anonSigner implements the OAuthSigner interface.
var _ OAuthSigner = anonSigner{}

// NewAnonymousClient creates a client that issues anonymous requests.
func NewAnonymousClient(BaseURL string) (*Client, error) {
	parsedBaseURL, err := url.Parse(BaseURL)
	if err != nil {
		return nil, err
	}
	return &Client{Signer: &anonSigner{}, BaseURL: parsedBaseURL}, nil
}

// NewAuthenticatedClient parses the given MAAS API key into the individual
// OAuth tokens and creates an Client that will use these tokens to sign the
// requests it issues.
func NewAuthenticatedClient(BaseURL string, apiKey string) (*Client, error) {
	elements := strings.Split(apiKey, ":")
	if len(elements) != 3 {
		errString := "invalid API key %q; expected \"<consumer secret>:<token key>:<token secret>\""
		return nil, fmt.Errorf(errString, apiKey)
	}
	token := &OAuthToken{
		ConsumerKey: elements[0],
		// The consumer secret is the empty string in MAAS' authentication.
		ConsumerSecret: "",
		TokenKey:       elements[1],
		TokenSecret:    elements[2],
	}
	signer, err := NewPlainTestOAuthSigner(token, "MAAS API")
	if err != nil {
		return nil, err
	}
	parsedBaseURL, err := url.Parse(BaseURL)
	if err != nil {
		return nil, err
	}
	return &Client{Signer: signer, BaseURL: parsedBaseURL}, nil
}
