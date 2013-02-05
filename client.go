// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"errors"
	"fmt"
	"io/ioutil"
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

func (client Client) GetURL(uri *url.URL) *url.URL {
	return client.BaseURL.ResolveReference(uri)
}

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

// nonIdempotentRequest is a utility method to issue a PUT or a POST request.
func (client Client) nonIdempotentRequest(method string, uri *url.URL, parameters url.Values) ([]byte, error) {
	url := client.GetURL(uri)
	request, err := http.NewRequest(method, url.String(), strings.NewReader(string(parameters.Encode())))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return client.dispatchRequest(request)
}

func (client Client) Post(uri *url.URL, operation string, parameters url.Values) ([]byte, error) {
	queryParams := url.Values{"op": {operation}}
	uri.RawQuery = queryParams.Encode()
	return client.nonIdempotentRequest("POST", uri, parameters)
}

func (client Client) Put(uri *url.URL, parameters url.Values) ([]byte, error) {
	return client.nonIdempotentRequest("PUT", uri, parameters)
}

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

type anonSigner struct{}

func (signer anonSigner) OAuthSign(request *http.Request) error {
	return nil
}

// Trick to ensure *anonSigner implements the OAuthSigner interface.
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
	// The consumer secret is the empty string in MAAS' authentication.
	token := &OAuthToken{
		ConsumerKey:    elements[0],
		ConsumerSecret: "",
		TokenKey:       elements[1],
		TokenSecret:    elements[2],
	}
	signer, err := NewPLAINTEXTOAuthSigner(token, "MAAS API")
	if err != nil {
		return nil, err
	}
	parsedBaseURL, err := url.Parse(BaseURL)
	if err != nil {
		return nil, err
	}
	return &Client{Signer: signer, BaseURL: parsedBaseURL}, nil
}
