// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type Client interface {
	Get(URL string, parameters url.Values) (response []byte, err error)
	Post(URL string, parameters url.Values) (response []byte, err error)
	Put(URL string, parameters url.Values) (response []byte, err error)
	Delete(URL string, parameters url.Values) error
}

type genericClient struct{}

func (client *genericClient) Get(URL string, parameters url.Values) ([]byte, error) {
	// TODO: do a proper url.join here.
	queryUrl := URL + parameters.Encode()
	request, err := http.NewRequest("GET", queryUrl, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	client.Sign(request)
	httpClient := http.Client{}
	response, reqErr := httpClient.Do(request)
	if reqErr != nil {
		log.Println(reqErr)
		return nil, reqErr
	}

	body, parseErr := ioutil.ReadAll(response.Body)
	if parseErr != nil {
		log.Println(parseErr)
		return nil, parseErr
	}
	return body, nil
}

func (client *genericClient) Post(URL string, parameters url.Values) ([]byte, error) {
	// Not implemented.
	return []byte{}, nil
}
func (client *genericClient) Put(URL string, parameters url.Values) ([]byte, error) {
	// Not implemented.
	return []byte{}, nil
}
func (client *genericClient) Delete(URL string, parameters url.Values) error {
	// Not implemented.
	return nil
}

// Trick to ensure *genericClient implements the Client interface.
var _ Client = (*genericClient)(nil)

// Sign does not do anything but is here to let children implement it.
func (client *genericClient) Sign(request *http.Request) error {
	return nil
}

// AnonymousClient implements a client which performs non-authenticated
// requests.
type AnonymousClient struct {
	genericClient
}

// AuthenticatedClient implements a client which performs OAuth-authenticated
// requests.
type AuthenticatedClient struct {
	genericClient
	consumerKey    string
	consumerSecret string
	tokenKey       string
	tokenSecret    string
}

func NewAuthenticatedClient(apiKey string) *AuthenticatedClient {
	// Parse MAAS API key and create an OAuthClient.
	// Not implemented.
	return nil
}

func (client *AuthenticatedClient) Sign(request *http.Request) error {
	// Sign the request with OAuth signature.
	// Not implemented.
	return nil
}
