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

// checkError is a shorthand helper that panics if err is not nil.
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// NewTestMAAS returns a TestMAASObject that implements the MAASObject
// interface and thus can be used as a test object instead of the one returned
// by gomaasapi.NewMAAS().
func NewTestMAAS(version string) *TestMAASObject {
	server := NewTestServer(version)
	authClient, err := NewAnonymousClient(server.URL, version)
	checkError(err)
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
	serveMux   *http.ServeMux
	client     Client
	nodes      map[string]MAASObject
	ownedNodes map[string]bool
	// mapping system_id -> list of operations performed.
	nodeOperations map[string][]string
	// mapping system_id -> list of Values passed when performing
	// operations
	nodeOperationRequestValues map[string][]url.Values
	files                      map[string]MAASObject
	version                    string
}

func getNodeURI(version, systemId string) string {
	return fmt.Sprintf("/api/%s/nodes/%s/", version, systemId)
}

func getFileURI(version, filename string) string {
	uri := url.URL{}
	uri.Path = fmt.Sprintf("/api/%s/files/%s/", version, filename)
	return uri.String()
}

// Clear clears all the fake data stored and recorded by the test server
// (nodes, recorded operations, etc.).
func (server *TestServer) Clear() {
	server.nodes = make(map[string]MAASObject)
	server.ownedNodes = make(map[string]bool)
	server.nodeOperations = make(map[string][]string)
	server.nodeOperationRequestValues = make(map[string][]url.Values)
	server.files = make(map[string]MAASObject)
}

// NodeOperations returns the map containing the list of the operations
// performed for each node.
func (server *TestServer) NodeOperations() map[string][]string {
	return server.nodeOperations
}

// NodeOperationRequestValues returns the map containing the list of the
// url.Values extracted from the request used when performing operations
// on nodes.
func (server *TestServer) NodeOperationRequestValues() map[string][]url.Values {
	return server.nodeOperationRequestValues
}

func (server *TestServer) addNodeOperation(systemId, operation string, request *http.Request) {
	operations, present := server.nodeOperations[systemId]
	operationRequestValues, present2 := server.nodeOperationRequestValues[systemId]
	if present != present2 {
		panic("inconsistent state: nodeOperations and nodeOperationRequestValues don't have the same keys.")
	}
	requestValues := url.Values{}
	if request.Body != nil && request.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		body, err := readAndClose(request.Body)
		if err != nil {
			panic(err)
		}
		requestValues, err = url.ParseQuery(string(body))
		if err != nil {
			panic(err)
		}
	}
	if !present {
		operations = []string{operation}
		operationRequestValues = []url.Values{requestValues}
	} else {
		operations = append(operations, operation)
		operationRequestValues = append(operationRequestValues, requestValues)
	}
	server.nodeOperations[systemId] = operations
	server.nodeOperationRequestValues[systemId] = operationRequestValues
}

// NewNode creates a MAAS node.  The provided string should be a valid json
// string representing a map and contain a string value for the key
// 'system_id'.  e.g. `{"system_id": "mysystemid"}`.
// If one of these conditions is not met, NewNode panics.
func (server *TestServer) NewNode(jsonText string) MAASObject {
	var attrs map[string]interface{}
	err := json.Unmarshal([]byte(jsonText), &attrs)
	checkError(err)
	systemIdEntry, hasSystemId := attrs["system_id"]
	if !hasSystemId {
		panic("The given map json string does not contain a 'system_id' value.")
	}
	systemId := systemIdEntry.(string)
	attrs[resourceURI] = getNodeURI(server.version, systemId)
	if _, hasStatus := attrs["status"]; !hasStatus {
		attrs["status"] = NodeStatusAllocated
	}
	obj := newJSONMAASObject(attrs, server.client)
	server.nodes[systemId] = obj
	return obj
}

// Nodes returns a map associating all the nodes' system ids with the nodes'
// objects.
func (server *TestServer) Nodes() map[string]MAASObject {
	return server.nodes
}

// OwnedNodes returns a map whose keys represent the nodes that are currently
// allocated.
func (server *TestServer) OwnedNodes() map[string]bool {
	return server.ownedNodes
}

// NewFile creates a file in the test MAAS server.
func (server *TestServer) NewFile(filename string, filecontent []byte) MAASObject {
	attrs := make(map[string]interface{})
	attrs[resourceURI] = getFileURI(server.version, filename)
	base64Content := base64.StdEncoding.EncodeToString(filecontent)
	attrs["content"] = base64Content
	attrs["filename"] = filename

	// Allocate an arbitrary URL here.  It would be nice if the caller
	// could do this, but that would change the API and require many
	// changes.
	escapedName := url.QueryEscape(filename)
	attrs["anon_resource_uri"] = "/maas/1.0/files/?op=get_by_key&key=" + escapedName + "_key"

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

func getTopLevelNodesURL(version string) string {
	return fmt.Sprintf("/api/%s/nodes/", version)
}

func getNodeURLRE(version string) *regexp.Regexp {
	reString := fmt.Sprintf("^/api/%s/nodes/([^/]*)/$", regexp.QuoteMeta(version))
	return regexp.MustCompile(reString)
}

func getFilesURL(version string) string {
	return fmt.Sprintf("/api/%s/files/", version)
}

func getFileURLRE(version string) *regexp.Regexp {
	reString := fmt.Sprintf("^/api/%s/files/(.*)/$", regexp.QuoteMeta(version))
	return regexp.MustCompile(reString)
}

// NewTestServer starts and returns a new MAAS test server. The caller should call Close when finished, to shut it down.
func NewTestServer(version string) *TestServer {
	server := &TestServer{version: version}

	serveMux := http.NewServeMux()
	nodesURL := getTopLevelNodesURL(server.version)
	// Register handler for '/api/<version>/nodes/*'.
	serveMux.HandleFunc(nodesURL, func(w http.ResponseWriter, r *http.Request) {
		nodesHandler(server, w, r)
	})
	filesURL := getFilesURL(server.version)
	// Register handler for '/api/<version>/files/*'.
	serveMux.HandleFunc(filesURL, func(w http.ResponseWriter, r *http.Request) {
		filesHandler(server, w, r)
	})

	newServer := httptest.NewServer(serveMux)
	client, err := NewAnonymousClient(newServer.URL, "1.0")
	checkError(err)
	server.Server = newServer
	server.serveMux = serveMux
	server.client = *client
	server.Clear()
	return server
}

// nodesHandler handles requests for '/api/<version>/nodes/*'.
func nodesHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	checkError(err)
	op := values.Get("op")
	nodeURLRE := getNodeURLRE(server.version)
	nodeURLMatch := nodeURLRE.FindStringSubmatch(r.URL.Path)
	nodesURL := getTopLevelNodesURL(server.version)
	switch {
	case r.URL.Path == nodesURL:
		nodesTopLevelHandler(server, w, r, op)
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
			server.addNodeOperation(systemId, operation, r)

			if operation == "release" {
				delete(server.OwnedNodes(), systemId)
			}

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
	values, err := url.ParseQuery(r.URL.RawQuery)
	checkError(err)
	ids, hasId := values["id"]
	var convertedNodes = []map[string]JSONObject{}
	for systemId, node := range server.nodes {
		if !hasId || contains(ids, systemId) {
			convertedNodes = append(convertedNodes, node.GetMap())
		}
	}
	res, err := json.Marshal(convertedNodes)
	checkError(err)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(res))
}

// findFreeNode looks for a node that is currently available.
func findFreeNode(server *TestServer) *MAASObject {
	for systemID, node := range server.Nodes() {
		_, present := server.OwnedNodes()[systemID]
		if !present {
			return &node
		}
	}
	return nil
}

// nodesAcquireHandler simulates acquiring a node.
func nodesAcquireHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	node := findFreeNode(server)
	if node == nil {
		w.WriteHeader(http.StatusConflict)
	} else {
		systemId, err := node.GetField("system_id")
		checkError(err)
		server.OwnedNodes()[systemId] = true
		res, err := json.Marshal(node)
		checkError(err)
		// Record operation.
		server.addNodeOperation(systemId, "acquire", r)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(res))
	}
}

// nodesTopLevelHandler handles a request for /api/<version>/nodes/
// (with no node id following as part of the path).
func nodesTopLevelHandler(server *TestServer, w http.ResponseWriter, r *http.Request, op string) {
	switch {
	case r.Method == "GET" && op == "list":
		// Node listing operation.
		nodeListingHandler(server, w, r)
	case r.Method == "POST" && op == "acquire":
		nodesAcquireHandler(server, w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

// filesHandler handles requests for '/api/<version>/files/*'.
func filesHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	checkError(err)
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

// listFilenames returns the names of those uploaded files whose names start
// with the given prefix, sorted lexicographically.
func listFilenames(server *TestServer, prefix string) []string {
	var filenames = make([]string, 0)
	for filename := range server.files {
		if strings.HasPrefix(filename, prefix) {
			filenames = append(filenames, filename)
		}
	}
	sort.Strings(filenames)
	return filenames
}

// stripFileContent copies a map of attributes representing an uploaded file,
// but with the "content" attribute removed.
func stripContent(original map[string]JSONObject) map[string]JSONObject {
	newMap := make(map[string]JSONObject, len(original)-1)
	for key, value := range original {
		if key != "content" {
			newMap[key] = value
		}
	}
	return newMap
}

// fileListingHandler handles requests for '/api/<version>/files/?op=list'.
func fileListingHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	checkError(err)
	prefix := values.Get("prefix")
	filenames := listFilenames(server, prefix)

	// Build a sorted list of the files as map[string]JSONObject objects.
	convertedFiles := make([]map[string]JSONObject, 0)
	for _, filename := range filenames {
		// The "content" attribute is not in the listing.
		fileMap := stripContent(server.files[filename].GetMap())
		convertedFiles = append(convertedFiles, fileMap)
	}
	res, err := json.Marshal(convertedFiles)
	checkError(err)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(res))
}

// fileHandler handles requests for '/api/<version>/files/<filename>/'.
func fileHandler(server *TestServer, w http.ResponseWriter, r *http.Request, filename string, operation string) {
	switch {
	case r.Method == "DELETE":
		delete(server.files, filename)
		w.WriteHeader(http.StatusOK)
	case r.Method == "GET":
		// Retrieve a file's information (including content) as a JSON
		// object.
		file, ok := server.files[filename]
		if !ok {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		jsonText, err := json.Marshal(file)
		if err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonText)
	default:
		// Default handler: not found.
		http.NotFoundHandler().ServeHTTP(w, r)
	}
}

// InternalError replies to the request with an HTTP 500 internal error.
func InternalError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// getFileHandler handles requests for
// '/api/<version>/files/?op=get&filename=filename'.
func getFileHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	checkError(err)
	filename := values.Get("filename")
	file, found := server.files[filename]
	if !found {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	base64Content, err := file.GetField("content")
	if err != nil {
		InternalError(w, r, err)
		return
	}
	content, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		InternalError(w, r, err)
		return
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
	checkError(err)

	filename := r.Form.Get("filename")
	if filename == "" {
		panic("upload has no filename")
	}

	uploads := r.MultipartForm.File
	if len(uploads) != 1 {
		panic("the payload should contain one file and one file only")
	}
	var upload *multipart.FileHeader
	for _, uploadContent := range uploads {
		upload = uploadContent[0]
	}
	content, err := readMultipart(upload)
	checkError(err)
	server.NewFile(filename, content)
	w.WriteHeader(http.StatusOK)
}
