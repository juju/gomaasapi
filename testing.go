// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

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
