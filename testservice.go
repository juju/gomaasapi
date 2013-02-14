// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// TestMAASObject is a fake MAAS server MAASObject.
type TestMAASObject struct {
	MAASObject
	TestServer *TestServer
}

// NewTestMAAS returns a TestMAASObject that implements the MAASObject
// interface and thus can be used as a test object instead of the one returned
// by gomaasapi.NewMAAS().
func NewTestMAAS(version string) *TestMAASObject {
	server := NewTestServer(version)
	authClient, _ := NewAnonymousClient(server.URL + fmt.Sprintf("/api/%s/", version))
	maas := NewMAAS(*authClient)
	return &TestMAASObject{*maas, server}
}

// Close shuts down the test server.
func (testMAASObject *TestMAASObject) Close() {
	testMAASObject.TestServer.Close()
}

// A TestServer is an HTTP server listening on a system-chosen port on the
// local loopback interface, which simulates the behavior of a MAAS server.
// It is intendend for use in end-to-end HTTP tests using the gomaasapi
// library.
type TestServer struct {
	*httptest.Server
	serveMux       *http.ServeMux
	client         Client
	nodes          map[string]MAASObject
	nodeOperations map[string][]string
	files          map[string]MAASObject
	version        string
}

func getNodeURI(version, systemId string) string {
	return fmt.Sprintf("/api/%s/nodes/%s/", version, systemId)
}

func getFileURI(version, filename string) string {
	return fmt.Sprintf("/api/%s/files/%s/", version, filename)
}

// Clear clears all the fake data stored and recorded by the test server
// (nodes, recorded operations, etc.).
func (server *TestServer) Clear() {
	server.nodes = make(map[string]MAASObject)
	server.nodeOperations = make(map[string][]string)
	server.files = make(map[string]MAASObject)
}

// NodeOperations returns the map containing the list of the operations
// performed for each node.
func (server *TestServer) NodeOperations() map[string][]string {
	return server.nodeOperations
}

func (server *TestServer) addNodeOperation(systemId, operation string) {
	operations, present := server.nodeOperations[systemId]
	if !present {
		operations = []string{operation}
	} else {
		operations = append(operations, operation)
	}
	server.nodeOperations[systemId] = operations
}

// NewNode creates a MAAS node.  The provided string should be a valid json
// string representing a map and contain a string value for the key 
// 'system_id'.  e.g. `{"system_id": "mysystemid"}`.
// If one of these conditions is not met, NewNode panics.
func (server *TestServer) NewNode(jsonText string) MAASObject {
	var attrs map[string]interface{}
	err := json.Unmarshal([]byte(jsonText), &attrs)
	if err != nil {
		panic(err)
	}
	systemIdEntry, hasSystemId := attrs["system_id"]
	if !hasSystemId {
		panic("The given map json string does not contain a 'system_id' value.")
	}
	systemId := systemIdEntry.(string)
	attrs[resourceURI] = getNodeURI(server.version, systemId)
	obj := newJSONMAASObject(attrs, server.client)
	server.nodes[systemId] = obj
	return obj
}

// Returns a map associating all the nodes' system ids with the nodes'
// objects.
func (server *TestServer) Nodes() map[string]MAASObject {
	return server.nodes
}

// NewFile creates a file in the test MAAS server.
func (server *TestServer) NewFile(filename string, filecontent []byte) MAASObject {
	attrs := make(map[string]interface{})
	attrs[resourceURI] = getFileURI(server.version, filename)
	base64Content := base64.StdEncoding.EncodeToString(filecontent)
	attrs["content"] = base64Content
	attrs["filename"] = filename
	obj := newJSONMAASObject(attrs, server.client)
	server.files[filename] = obj
	return obj
}

func (server *TestServer) Files() map[string]MAASObject {
	return server.files
}

// ChangeNode updates a node with the given key/value.
func (server *TestServer) ChangeNode(systemId, key, value string) {
	node, found := server.nodes[systemId]
	if !found {
		panic("No node with such 'system_id'.")
	}
	node.GetMap()[key] = maasify(server.client, value)
}

func getNodeListingURL(version string) string {
	return fmt.Sprintf("/api/%s/nodes/", version)
}

func getNodeURLRE(version string) *regexp.Regexp {
	reString := fmt.Sprintf("^/api/%s/nodes/([^/]*)/$", version)
	return regexp.MustCompile(reString)
}

func getFilesURL(version string) string {
	return fmt.Sprintf("/api/%s/files/", version)
}

func getFileURLRE(version string) *regexp.Regexp {
	reString := fmt.Sprintf("^/api/%s/files/([^/]*)/$", version)
	return regexp.MustCompile(reString)
}

// NewTestServer starts and returns a new MAAS test server. The caller should call Close when finished, to shut it down.
func NewTestServer(version string) *TestServer {
	server := &TestServer{version: version}

	serveMux := http.NewServeMux()
	nodeListingURL := getNodeListingURL(server.version)
	// Register handler for '/api/<version>/nodes/*'.
	serveMux.HandleFunc(nodeListingURL, func(w http.ResponseWriter, r *http.Request) {
		nodesHandler(server, w, r)
	})
	filesURL := getFilesURL(server.version)
	// Register handler for '/api/<version>/files/*'.
	serveMux.HandleFunc(filesURL, func(w http.ResponseWriter, r *http.Request) {
		filesHandler(server, w, r)
	})

	newServer := httptest.NewServer(serveMux)
	client, _ := NewAnonymousClient(newServer.URL)
	server.Server = newServer
	server.serveMux = serveMux
	server.client = *client
	server.Clear()
	return server
}

// nodesHandler handles requests for '/api/<version>/nodes/*'.
func nodesHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, _ := url.ParseQuery(r.URL.RawQuery)
	op := values.Get("op")
	nodeURLRE := getNodeURLRE(server.version)
	nodeURLMatch := nodeURLRE.FindStringSubmatch(r.URL.Path)
	nodeListingURL := getNodeListingURL(server.version)
	switch {
	case r.Method == "GET" && op == "list" && r.URL.Path == nodeListingURL:
		// Node listing operation.
		nodeListingHandler(server, w, r)
	case nodeURLMatch != nil:
		// Request for a single node.
		nodeHandler(server, w, r, nodeURLMatch[1], op)
	default:
		// Default handler: not found.
		http.NotFoundHandler().ServeHTTP(w, r)
	}
}

// nodeHandler handles requests for '/api/<version>/nodes/<system_id>/'.
func nodeHandler(server *TestServer, w http.ResponseWriter, r *http.Request, systemId string, operation string) {
	node, ok := server.nodes[systemId]
	if !ok {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	if r.Method == "GET" {
		if operation == "" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, marshalNode(node))
			return
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	if r.Method == "POST" {
		// The only operations supported are "start", "stop" and "release".
		if operation == "start" || operation == "stop" || operation == "release" {
			// Record operation on node.
			server.addNodeOperation(systemId, operation)

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, marshalNode(node))
			return
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	if r.Method == "DELETE" {
		delete(server.nodes, systemId)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.NotFoundHandler().ServeHTTP(w, r)
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// nodeListingHandler handles requests for '/nodes/'.
func nodeListingHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, _ := url.ParseQuery(r.URL.RawQuery)
	ids, hasId := values["id"]
	var convertedNodes = []map[string]JSONObject{}
	for systemId, node := range server.nodes {
		if !hasId || contains(ids, systemId) {
			convertedNodes = append(convertedNodes, node.GetMap())
		}
	}
	res, _ := json.Marshal(convertedNodes)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(res))
}

// filesHandler handles requests for '/api/<version>/files/*'.
func filesHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, _ := url.ParseQuery(r.URL.RawQuery)
	op := values.Get("op")
	fileURLRE := getFileURLRE(server.version)
	fileURLMatch := fileURLRE.FindStringSubmatch(r.URL.Path)
	fileListingURL := getFilesURL(server.version)
	switch {
	case r.Method == "GET" && op == "list" && r.URL.Path == fileListingURL:
		// File listing operation.
		fileListingHandler(server, w, r)
	case op == "get" && r.Method == "GET" && r.URL.Path == fileListingURL:
		getFileHandler(server, w, r)
	case op == "add" && r.Method == "POST" && r.URL.Path == fileListingURL:
		addFileHandler(server, w, r)
	case fileURLMatch != nil:
		// Request for a single file.
		fileHandler(server, w, r, fileURLMatch[1], op)
	default:
		// Default handler: not found.
		http.NotFoundHandler().ServeHTTP(w, r)
	}

}

// fileListingHandler handles requests for '/api/<version>/files/?op=list'.
func fileListingHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, _ := url.ParseQuery(r.URL.RawQuery)
	prefix := values.Get("prefix")
	var convertedFiles = []map[string]JSONObject{}
	// Create slice of selected filenames.
	var filenames = []string{}
	for filename, _ := range server.files {
		if strings.Index(filename, prefix) == 0 {
			filenames = append(filenames, filename)
		}
	}
	// Sort filenames.
	sort.Strings(filenames)
	// Build result. 
	for _, filename := range filenames {
		file, _ := server.files[filename]
		fileMap := file.GetMap()
		delete(fileMap, "content")
		convertedFiles = append(convertedFiles, fileMap)
	}
	res, _ := json.Marshal(convertedFiles)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(res))
}

// fileHandler handles requests for '/api/<version>/files/<filename>/'.
func fileHandler(server *TestServer, w http.ResponseWriter, r *http.Request, filename string, operation string) {
	switch {
	case r.Method == "DELETE":
		delete(server.files, filename)
		w.WriteHeader(http.StatusOK)
		return
	default:
		// Default handler: not found.
		http.NotFoundHandler().ServeHTTP(w, r)
	}
}

// getFileHandler handles requests for
// '/api/<version>/files/?op=get&filename=filename'.
func getFileHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, _ := url.ParseQuery(r.URL.RawQuery)
	filename := values.Get("filename")
	file, found := server.files[filename]
	if !found {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	base64Content, err := file.GetField("content")
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, r)
	}
	content, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, r)
	}
	w.Write(content)
}

func readMultipart(upload *multipart.FileHeader) ([]byte, error) {
	file, err := upload.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	return ioutil.ReadAll(reader)
}

// filesHandler handles requests for '/api/<version>/files/?op=add&filename=filename'.
func addFileHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10000000)
	if err != nil {
		panic(err)
	}

	values, _ := url.ParseQuery(r.URL.RawQuery)
	filename := values.Get("filename")

	uploads := r.MultipartForm.File
	if len(uploads) != 1 {
		panic("the payload should contain one file and one file only")
	}
	var upload *multipart.FileHeader
	for _, uploadContent := range uploads {
		upload = uploadContent[0]
	}
	content, err := readMultipart(upload)
	if err != nil {
		panic(err)
	}
	server.NewFile(filename, content)
	w.WriteHeader(http.StatusOK)
}
