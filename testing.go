// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

type singleServingServer struct {
	*httptest.Server
	requestContent *string
	requestHeader  *http.Header
}

// newSingleServingServer creates a single-serving test http server which will
// return only one response as defined by the passed arguments.
func newSingleServingServer(uri string, response string, code int) *singleServingServer {
	var requestContent string
	var requestHeader http.Header
	var requested bool
	handler := func(writer http.ResponseWriter, request *http.Request) {
		if requested {
			http.Error(writer, "Already requested", http.StatusServiceUnavailable)
		}
		res, err := readAndClose(request.Body)
		if err != nil {
			panic(err)
		}
		requestContent = string(res)
		requestHeader = request.Header
		if request.URL.String() != uri {
			errorMsg := fmt.Sprintf("Error 404: page not found (expected '%v', got '%v').", uri, request.URL.String())
			http.Error(writer, errorMsg, http.StatusNotFound)
		} else {
			writer.WriteHeader(code)
			fmt.Fprint(writer, response)
		}
		requested = true
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	return &singleServingServer{server, &requestContent, &requestHeader}
}

type flakyServer struct {
	*httptest.Server
	nbRequests *int
	requests   *[][]byte
}

// newFlakyServer creates a "flaky" test http server which will
// return `nbFlakyResponses` responses with the given code and then a 200 response.
func newFlakyServer(uri string, code int, nbFlakyResponses int) *flakyServer {
	nbRequests := 0
	requests := make([][]byte, nbFlakyResponses+1)
	handler := func(writer http.ResponseWriter, request *http.Request) {
		nbRequests += 1
		body, err := readAndClose(request.Body)
		if err != nil {
			panic(err)
		}
		requests[nbRequests-1] = body
		if request.URL.String() != uri {
			errorMsg := fmt.Sprintf("Error 404: page not found (expected '%v', got '%v').", uri, request.URL.String())
			http.Error(writer, errorMsg, http.StatusNotFound)
		} else if nbRequests <= nbFlakyResponses {
			if code == http.StatusServiceUnavailable {
				writer.Header().Set("Retry-After", "0")
			}
			writer.WriteHeader(code)
			fmt.Fprint(writer, "flaky")
		} else {
			writer.WriteHeader(http.StatusOK)
			fmt.Fprint(writer, "ok")
		}

	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	return &flakyServer{server, &nbRequests, &requests}
}

type simpleResponse struct {
	status int
	body   string
}

type SimpleTestServer struct {
	*httptest.Server

	getResponses        map[string][]simpleResponse
	getResponseIndex    map[string]int
	postResponses       map[string][]simpleResponse
	postResponseIndex   map[string]int
	deleteResponses     map[string][]simpleResponse
	deleteResponseIndex map[string]int

	requests []*http.Request
}

func NewSimpleServer() *SimpleTestServer {
	server := &SimpleTestServer{
		getResponses:        make(map[string][]simpleResponse),
		getResponseIndex:    make(map[string]int),
		postResponses:       make(map[string][]simpleResponse),
		postResponseIndex:   make(map[string]int),
		deleteResponses:     make(map[string][]simpleResponse),
		deleteResponseIndex: make(map[string]int),
	}
	server.Server = httptest.NewUnstartedServer(http.HandlerFunc(server.handler))
	return server
}

func (s *SimpleTestServer) AddGetResponse(path string, status int, body string) {
	s.getResponses[path] = append(s.getResponses[path], simpleResponse{status: status, body: body})
}

func (s *SimpleTestServer) AddPostResponse(path string, status int, body string) {
	s.postResponses[path] = append(s.postResponses[path], simpleResponse{status: status, body: body})
}

func (s *SimpleTestServer) AddDeleteResponse(path string, status int, body string) {
	s.deleteResponses[path] = append(s.deleteResponses[path], simpleResponse{status: status, body: body})
}

func (s *SimpleTestServer) LastRequest() *http.Request {
	pos := len(s.requests) - 1
	if pos < 0 {
		return nil
	}
	return s.requests[pos]
}

func (s *SimpleTestServer) handler(writer http.ResponseWriter, request *http.Request) {
	method := request.Method
	var responses map[string][]simpleResponse
	var responseIndex map[string]int
	switch method {
	case "GET":
		responses = s.getResponses
		responseIndex = s.getResponseIndex
		_, err := readAndClose(request.Body)
		if err != nil {
			panic(err) // it is a test, panic should be fine
		}
	case "POST":
		responses = s.postResponses
		responseIndex = s.postResponseIndex
		err := request.ParseForm()
		if err != nil {
			panic(err)
		}
	case "DELETE":
		responses = s.deleteResponses
		responseIndex = s.deleteResponseIndex
		_, err := readAndClose(request.Body)
		if err != nil {
			panic(err)
		}
	default:
		panic("unsupported method " + method)
	}
	s.requests = append(s.requests, request)
	uri := request.URL.String()
	testResponses, found := responses[uri]
	if !found {
		errorMsg := fmt.Sprintf("Error 404: page not found ('%v').", uri)
		http.Error(writer, errorMsg, http.StatusNotFound)
	} else {
		index := responseIndex[uri]
		response := testResponses[index]
		responseIndex[uri] = index + 1

		writer.WriteHeader(response.status)
		fmt.Fprint(writer, response.body)
	}
}
