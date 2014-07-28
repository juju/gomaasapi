// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"gopkg.in/mgo.v2/bson"
	. "launchpad.net/gocheck"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type TestServerSuite struct {
	server *TestServer
}

var _ = Suite(&TestServerSuite{})

func (suite *TestServerSuite) SetUpTest(c *C) {
	server := NewTestServer("1.0")
	suite.server = server
}

func (suite *TestServerSuite) TearDownTest(c *C) {
	suite.server.Close()
}

func (suite *TestServerSuite) TestNewTestServerReturnsTestServer(c *C) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}
	suite.server.serveMux.HandleFunc("/test/", handler)
	resp, err := http.Get(suite.server.Server.URL + "/test/")

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusAccepted)
}

func (suite *TestServerSuite) TestGetResourceURI(c *C) {
	c.Check(getNodeURL("0.1", "test"), Equals, "/api/0.1/nodes/test/")
}

func (suite *TestServerSuite) TestInvalidOperationOnNodesIsBadRequest(c *C) {
	badURL := getNodesEndpoint(suite.server.version) + "?op=procrastinate"

	response, err := http.Get(suite.server.Server.URL + badURL)
	c.Assert(err, IsNil)

	c.Check(response.StatusCode, Equals, http.StatusBadRequest)
}

func (suite *TestServerSuite) TestHandlesNodeListingUnknownPath(c *C) {
	invalidPath := fmt.Sprintf("/api/%s/nodes/invalid/path/", suite.server.version)
	resp, err := http.Get(suite.server.Server.URL + invalidPath)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *TestServerSuite) TestNewNode(c *C) {
	input := `{"system_id": "mysystemid"}`

	newNode := suite.server.NewNode(input)

	c.Check(len(suite.server.nodes), Equals, 1)
	c.Check(suite.server.nodes["mysystemid"], DeepEquals, newNode)
}

func (suite *TestServerSuite) TestNodesReturnsNodes(c *C) {
	input := `{"system_id": "mysystemid"}`
	newNode := suite.server.NewNode(input)

	nodesMap := suite.server.Nodes()

	c.Check(len(nodesMap), Equals, 1)
	c.Check(nodesMap["mysystemid"], DeepEquals, newNode)
}

func (suite *TestServerSuite) TestChangeNode(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	suite.server.ChangeNode("mysystemid", "newfield", "newvalue")

	node, _ := suite.server.nodes["mysystemid"]
	field, err := node.GetField("newfield")
	c.Assert(err, IsNil)
	c.Check(field, Equals, "newvalue")
}

func (suite *TestServerSuite) TestClearClearsData(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	suite.server.addNodeOperation("mysystemid", "start", &http.Request{})

	suite.server.Clear()

	c.Check(len(suite.server.nodes), Equals, 0)
	c.Check(len(suite.server.nodeOperations), Equals, 0)
	c.Check(len(suite.server.nodeOperationRequestValues), Equals, 0)
}

func (suite *TestServerSuite) TestAddNodeOperationPopulatesOperations(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)

	suite.server.addNodeOperation("mysystemid", "start", &http.Request{})
	suite.server.addNodeOperation("mysystemid", "stop", &http.Request{})

	nodeOperations := suite.server.NodeOperations()
	operations := nodeOperations["mysystemid"]
	c.Check(operations, DeepEquals, []string{"start", "stop"})
}

func (suite *TestServerSuite) TestAddNodeOperationPopulatesOperationRequestValues(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	reader := strings.NewReader("key=value")
	request, err := http.NewRequest("POST", "http://example.com/", reader)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Assert(err, IsNil)

	suite.server.addNodeOperation("mysystemid", "start", request)

	values := suite.server.NodeOperationRequestValues()
	value := values["mysystemid"]
	c.Check(len(value), Equals, 1)
	c.Check(value[0], DeepEquals, url.Values{"key": []string{"value"}})
}

func (suite *TestServerSuite) TestNewNodeRequiresJSONString(c *C) {
	input := `invalid:json`
	defer func() {
		recoveredError := recover().(*json.SyntaxError)
		c.Check(recoveredError, NotNil)
		c.Check(recoveredError.Error(), Matches, ".*invalid character.*")
	}()
	suite.server.NewNode(input)
}

func (suite *TestServerSuite) TestNewNodeRequiresSystemIdKey(c *C) {
	input := `{"test": "test"}`
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		c.Check(recoveredError, Matches, ".*does not contain a 'system_id' value.")
	}()
	suite.server.NewNode(input)
}

func (suite *TestServerSuite) TestHandlesNodeRequestNotFound(c *C) {
	getURI := fmt.Sprintf("/api/%s/nodes/test/", suite.server.version)
	resp, err := http.Get(suite.server.Server.URL + getURI)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *TestServerSuite) TestHandlesNodeUnknownOperation(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	postURI := fmt.Sprintf("/api/%s/nodes/mysystemid/?op=unknown/", suite.server.version)
	respStart, err := http.Post(suite.server.Server.URL+postURI, "", nil)

	c.Check(err, IsNil)
	c.Check(respStart.StatusCode, Equals, http.StatusBadRequest)
}

func (suite *TestServerSuite) TestHandlesNodeDelete(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	deleteURI := fmt.Sprintf("/api/%s/nodes/mysystemid/?op=mysystemid", suite.server.version)
	req, err := http.NewRequest("DELETE", suite.server.Server.URL+deleteURI, nil)
	var client http.Client
	resp, err := client.Do(req)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	c.Check(len(suite.server.nodes), Equals, 0)
}

func uploadTo(url, fileName string, fileContent []byte) (*http.Response, error) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	fw, err := w.CreateFormFile(fileName, fileName)
	if err != nil {
		panic(err)
	}
	io.Copy(fw, bytes.NewBuffer(fileContent))
	w.Close()
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{}
	return client.Do(req)
}

func (suite *TestServerSuite) TestHandlesUploadFile(c *C) {
	fileContent := []byte("test file content")
	postURL := suite.server.Server.URL + fmt.Sprintf("/api/%s/files/?op=add&filename=filename", suite.server.version)

	resp, err := uploadTo(postURL, "upload", fileContent)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	c.Check(len(suite.server.files), Equals, 1)
	file, ok := suite.server.files["filename"]
	c.Assert(ok, Equals, true)
	field, err := file.GetField("content")
	c.Assert(err, IsNil)
	c.Check(field, Equals, base64.StdEncoding.EncodeToString(fileContent))
}

func (suite *TestServerSuite) TestNewFileEscapesName(c *C) {
	obj := suite.server.NewFile("aa?bb", []byte("bytes"))
	resourceURI := obj.URI()
	c.Check(strings.Contains(resourceURI.String(), "aa?bb"), Equals, false)
	c.Check(strings.Contains(resourceURI.Path, "aa?bb"), Equals, true)
	anonURI, err := obj.GetField("anon_resource_uri")
	c.Assert(err, IsNil)
	c.Check(strings.Contains(anonURI, "aa?bb"), Equals, false)
	c.Check(strings.Contains(anonURI, url.QueryEscape("aa?bb")), Equals, true)
}

func (suite *TestServerSuite) TestHandlesFile(c *C) {
	const filename = "my-file"
	const fileContent = "test file content"
	file := suite.server.NewFile(filename, []byte(fileContent))
	getURI := fmt.Sprintf("/api/%s/files/%s/", suite.server.version, filename)
	fileURI, err := file.GetField("anon_resource_uri")
	c.Assert(err, IsNil)

	resp, err := http.Get(suite.server.Server.URL + getURI)
	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)

	content, err := readAndClose(resp.Body)
	c.Assert(err, IsNil)
	var obj map[string]interface{}
	err = json.Unmarshal(content, &obj)
	c.Assert(err, IsNil)
	anon_url, ok := obj["anon_resource_uri"]
	c.Check(ok, Equals, true)
	c.Check(anon_url.(string), Equals, fileURI)
	base64Content, ok := obj["content"]
	c.Check(ok, Equals, true)
	decodedContent, err := base64.StdEncoding.DecodeString(base64Content.(string))
	c.Assert(err, IsNil)
	c.Check(string(decodedContent), Equals, fileContent)
}

func (suite *TestServerSuite) TestHandlesGetFile(c *C) {
	fileContent := []byte("test file content")
	fileName := "filename"
	suite.server.NewFile(fileName, fileContent)
	getURI := fmt.Sprintf("/api/%s/files/?op=get&filename=filename", suite.server.version)

	resp, err := http.Get(suite.server.Server.URL + getURI)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	content, err := readAndClose(resp.Body)
	c.Check(err, IsNil)
	c.Check(string(content), Equals, string(fileContent))
	c.Check(content, DeepEquals, fileContent)
}

func (suite *TestServerSuite) TestHandlesListReturnsSortedFilenames(c *C) {
	fileName1 := "filename1"
	suite.server.NewFile(fileName1, []byte("test file content"))
	fileName2 := "filename2"
	suite.server.NewFile(fileName2, []byte("test file content"))
	getURI := fmt.Sprintf("/api/%s/files/?op=list", suite.server.version)

	resp, err := http.Get(suite.server.Server.URL + getURI)
	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	content, err := readAndClose(resp.Body)
	c.Assert(err, IsNil)
	var files []map[string]string
	err = json.Unmarshal(content, &files)
	c.Assert(err, IsNil)
	c.Check(len(files), Equals, 2)
	c.Check(files[0]["filename"], Equals, fileName1)
	c.Check(files[1]["filename"], Equals, fileName2)
}

func (suite *TestServerSuite) TestHandlesListFiltersFiles(c *C) {
	fileName1 := "filename1"
	suite.server.NewFile(fileName1, []byte("test file content"))
	fileName2 := "prefixFilename"
	suite.server.NewFile(fileName2, []byte("test file content"))
	getURI := fmt.Sprintf("/api/%s/files/?op=list&prefix=prefix", suite.server.version)

	resp, err := http.Get(suite.server.Server.URL + getURI)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	content, err := readAndClose(resp.Body)
	c.Assert(err, IsNil)
	var files []map[string]string
	err = json.Unmarshal(content, &files)
	c.Assert(err, IsNil)
	c.Check(len(files), Equals, 1)
	c.Check(files[0]["filename"], Equals, fileName2)
}

func (suite *TestServerSuite) TestHandlesListOmitsContent(c *C) {
	const filename = "myfile"
	fileContent := []byte("test file content")
	suite.server.NewFile(filename, fileContent)
	getURI := fmt.Sprintf("/api/%s/files/?op=list", suite.server.version)

	resp, err := http.Get(suite.server.Server.URL + getURI)
	c.Assert(err, IsNil)

	content, err := readAndClose(resp.Body)
	c.Assert(err, IsNil)
	var files []map[string]string
	err = json.Unmarshal(content, &files)

	// The resulting dict does not have a "content" entry.
	file := files[0]
	_, ok := file["content"]
	c.Check(ok, Equals, false)

	// But the original as stored in the test service still has it.
	contentAfter, err := suite.server.files[filename].GetField("content")
	c.Assert(err, IsNil)
	bytes, err := base64.StdEncoding.DecodeString(contentAfter)
	c.Assert(err, IsNil)
	c.Check(string(bytes), Equals, string(fileContent))
}

func (suite *TestServerSuite) TestDeleteFile(c *C) {
	fileName1 := "filename1"
	suite.server.NewFile(fileName1, []byte("test file content"))
	deleteURI := fmt.Sprintf("/api/%s/files/filename1/", suite.server.version)

	req, err := http.NewRequest("DELETE", suite.server.Server.URL+deleteURI, nil)
	c.Check(err, IsNil)
	var client http.Client
	resp, err := client.Do(req)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	c.Check(suite.server.Files(), DeepEquals, map[string]MAASObject{})
}

// TestMAASObjectSuite validates that the object created by
// NewTestMAAS can be used by the gomaasapi library as if it were a real
// MAAS server.
type TestMAASObjectSuite struct {
	TestMAASObject *TestMAASObject
}

var _ = Suite(&TestMAASObjectSuite{})

func (s *TestMAASObjectSuite) SetUpSuite(c *C) {
	s.TestMAASObject = NewTestMAAS("1.0")
}

func (s *TestMAASObjectSuite) TearDownSuite(c *C) {
	s.TestMAASObject.Close()
}

func (s *TestMAASObjectSuite) TearDownTest(c *C) {
	s.TestMAASObject.TestServer.Clear()
}

func (suite *TestMAASObjectSuite) TestListNodes(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")

	listNodeObjects, err := nodeListing.CallGet("list", url.Values{})

	c.Check(err, IsNil)
	listNodes, err := listNodeObjects.GetArray()
	c.Assert(err, IsNil)
	c.Check(len(listNodes), Equals, 1)
	node, err := listNodes[0].GetMAASObject()
	c.Assert(err, IsNil)
	systemId, err := node.GetField("system_id")
	c.Assert(err, IsNil)
	c.Check(systemId, Equals, "mysystemid")
	resourceURI, _ := node.GetField(resourceURI)
	apiVersion := suite.TestMAASObject.TestServer.version
	expectedResourceURI := fmt.Sprintf("/api/%s/nodes/mysystemid/", apiVersion)
	c.Check(resourceURI, Equals, expectedResourceURI)
}

func (suite *TestMAASObjectSuite) TestListNodesNoNodes(c *C) {
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	listNodeObjects, err := nodeListing.CallGet("list", url.Values{})
	c.Check(err, IsNil)

	listNodes, err := listNodeObjects.GetArray()

	c.Check(err, IsNil)
	c.Check(listNodes, DeepEquals, []JSONObject{})
}

func (suite *TestMAASObjectSuite) TestListNodesSelectedNodes(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	input2 := `{"system_id": "mysystemid2"}`
	suite.TestMAASObject.TestServer.NewNode(input2)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")

	listNodeObjects, err := nodeListing.CallGet("list", url.Values{"id": {"mysystemid2"}})

	c.Check(err, IsNil)
	listNodes, err := listNodeObjects.GetArray()
	c.Check(err, IsNil)
	c.Check(len(listNodes), Equals, 1)
	node, _ := listNodes[0].GetMAASObject()
	systemId, _ := node.GetField("system_id")
	c.Check(systemId, Equals, "mysystemid2")
}

func (suite *TestMAASObjectSuite) TestDeleteNode(c *C) {
	input := `{"system_id": "mysystemid"}`
	node := suite.TestMAASObject.TestServer.NewNode(input)

	err := node.Delete()

	c.Check(err, IsNil)
	c.Check(suite.TestMAASObject.TestServer.Nodes(), DeepEquals, map[string]MAASObject{})
}

func (suite *TestMAASObjectSuite) TestOperationsOnNode(c *C) {
	input := `{"system_id": "mysystemid"}`
	node := suite.TestMAASObject.TestServer.NewNode(input)
	operations := []string{"start", "stop", "release"}
	for _, operation := range operations {
		_, err := node.CallPost(operation, url.Values{})
		c.Check(err, IsNil)
	}
}

func (suite *TestMAASObjectSuite) TestOperationsOnNodeGetRecorded(c *C) {
	input := `{"system_id": "mysystemid"}`
	node := suite.TestMAASObject.TestServer.NewNode(input)

	_, err := node.CallPost("start", url.Values{})

	c.Check(err, IsNil)
	nodeOperations := suite.TestMAASObject.TestServer.NodeOperations()
	operations := nodeOperations["mysystemid"]
	c.Check(operations, DeepEquals, []string{"start"})
}

func (suite *TestMAASObjectSuite) TestAcquireOperationGetsRecorded(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodesObj := suite.TestMAASObject.GetSubObject("nodes/")
	params := url.Values{"key": []string{"value"}}

	jsonResponse, err := nodesObj.CallPost("acquire", params)
	c.Assert(err, IsNil)
	acquiredNode, err := jsonResponse.GetMAASObject()
	c.Assert(err, IsNil)
	systemId, err := acquiredNode.GetField("system_id")
	c.Assert(err, IsNil)

	// The 'acquire' operation has been recorded.
	nodeOperations := suite.TestMAASObject.TestServer.NodeOperations()
	operations := nodeOperations[systemId]
	c.Check(operations, DeepEquals, []string{"acquire"})

	// The parameters used to 'acquire' the node have been recorded as well.
	values := suite.TestMAASObject.TestServer.NodeOperationRequestValues()
	value := values[systemId]
	c.Check(len(value), Equals, 1)
	c.Check(value[0], DeepEquals, params)
}

func (suite *TestMAASObjectSuite) TestNodesRelease(c *C) {
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "mysystemid1"}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "mysystemid2"}`)
	suite.TestMAASObject.TestServer.OwnedNodes()["mysystemid2"] = true
	nodesObj := suite.TestMAASObject.GetSubObject("nodes/")
	params := url.Values{"nodes": []string{"mysystemid1", "mysystemid2"}}

	// release should only release mysystemid2, as it is the only one allocated.
	jsonResponse, err := nodesObj.CallPost("release", params)
	c.Assert(err, IsNil)
	releasedNodes, err := jsonResponse.GetArray()
	c.Assert(err, IsNil)
	c.Assert(releasedNodes, HasLen, 1)
	releasedNode, err := releasedNodes[0].GetMAASObject()
	c.Assert(err, IsNil)
	systemId, err := releasedNode.GetField("system_id")
	c.Assert(err, IsNil)
	c.Assert(systemId, Equals, "mysystemid2")

	// The 'release' operation has been recorded.
	nodesOperations := suite.TestMAASObject.TestServer.NodesOperations()
	c.Check(nodesOperations, DeepEquals, []string{"release"})
	nodesOperationRequestValues := suite.TestMAASObject.TestServer.NodesOperationRequestValues()
	expectedValues := make(url.Values)
	expectedValues.Add("nodes", "mysystemid1")
	expectedValues.Add("nodes", "mysystemid2")
	c.Check(nodesOperationRequestValues, DeepEquals, []url.Values{expectedValues})
}

func (suite *TestMAASObjectSuite) TestNodesReleaseUnknown(c *C) {
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "mysystemid"}`)
	suite.TestMAASObject.TestServer.OwnedNodes()["mysystemid"] = true
	nodesObj := suite.TestMAASObject.GetSubObject("nodes/")
	params := url.Values{"nodes": []string{"mysystemid", "what"}}

	// if there are any unknown nodes, none are released.
	_, err := nodesObj.CallPost("release", params)
	c.Assert(err, ErrorMatches, `gomaasapi: got error back from server: 400 Bad Request \(Unknown node\(s\): what.\)`)
	c.Assert(suite.TestMAASObject.TestServer.OwnedNodes()["mysystemid"], Equals, true)
}

func (suite *TestMAASObjectSuite) TestUploadFile(c *C) {
	const filename = "myfile.txt"
	const fileContent = "uploaded contents"
	files := suite.TestMAASObject.GetSubObject("files")
	params := url.Values{"filename": {filename}}
	filesMap := map[string][]byte{"file": []byte(fileContent)}

	// Upload a file.
	_, err := files.CallPostFiles("add", params, filesMap)
	c.Assert(err, IsNil)

	// The file can now be downloaded.
	downloadedFile, err := files.CallGet("get", params)
	c.Assert(err, IsNil)
	bytes, err := downloadedFile.GetBytes()
	c.Assert(err, IsNil)
	c.Check(string(bytes), Equals, fileContent)
}

func (suite *TestMAASObjectSuite) TestFileNamesMayContainSlashes(c *C) {
	const filename = "filename/with/slashes/in/it"
	const fileContent = "file contents"
	files := suite.TestMAASObject.GetSubObject("files")
	params := url.Values{"filename": {filename}}
	filesMap := map[string][]byte{"file": []byte(fileContent)}

	_, err := files.CallPostFiles("add", params, filesMap)
	c.Assert(err, IsNil)

	file, err := files.GetSubObject(filename).Get()
	c.Assert(err, IsNil)
	field, err := file.GetField("content")
	c.Assert(err, IsNil)
	c.Check(field, Equals, base64.StdEncoding.EncodeToString([]byte(fileContent)))
}

func (suite *TestMAASObjectSuite) TestAcquireNodeGrabsAvailableNode(c *C) {
	input := `{"system_id": "nodeid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodesObj := suite.TestMAASObject.GetSubObject("nodes/")

	jsonResponse, err := nodesObj.CallPost("acquire", nil)
	c.Assert(err, IsNil)

	acquiredNode, err := jsonResponse.GetMAASObject()
	c.Assert(err, IsNil)
	systemID, err := acquiredNode.GetField("system_id")
	c.Assert(err, IsNil)
	c.Check(systemID, Equals, "nodeid")
	_, owned := suite.TestMAASObject.TestServer.OwnedNodes()[systemID]
	c.Check(owned, Equals, true)
}

func (suite *TestMAASObjectSuite) TestAcquireNodeNeedsANode(c *C) {
	nodesObj := suite.TestMAASObject.GetSubObject("nodes/")
	_, err := nodesObj.CallPost("acquire", nil)
	c.Check(err.(ServerError).StatusCode, Equals, http.StatusConflict)
}

func (suite *TestMAASObjectSuite) TestAcquireNodeIgnoresOwnedNodes(c *C) {
	input := `{"system_id": "nodeid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodesObj := suite.TestMAASObject.GetSubObject("nodes/")
	// Ensure that the one node in the MAAS is not available.
	_, err := nodesObj.CallPost("acquire", nil)
	c.Assert(err, IsNil)

	_, err = nodesObj.CallPost("acquire", nil)
	c.Check(err.(ServerError).StatusCode, Equals, http.StatusConflict)
}

func (suite *TestMAASObjectSuite) TestReleaseNodeReleasesAcquiredNode(c *C) {
	input := `{"system_id": "nodeid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodesObj := suite.TestMAASObject.GetSubObject("nodes/")
	jsonResponse, err := nodesObj.CallPost("acquire", nil)
	c.Assert(err, IsNil)
	acquiredNode, err := jsonResponse.GetMAASObject()
	c.Assert(err, IsNil)
	systemID, err := acquiredNode.GetField("system_id")
	c.Assert(err, IsNil)
	nodeObj := nodesObj.GetSubObject(systemID)

	_, err = nodeObj.CallPost("release", nil)
	c.Assert(err, IsNil)
	_, owned := suite.TestMAASObject.TestServer.OwnedNodes()[systemID]
	c.Check(owned, Equals, false)
}

func (suite *TestMAASObjectSuite) TestGetNetworks(c *C) {
	nodeJSON := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(nodeJSON)
	networkJSON := `{"name": "mynetworkname"}`
	suite.TestMAASObject.TestServer.NewNetwork(networkJSON)
	suite.TestMAASObject.TestServer.ConnectNodeToNetwork("mysystemid", "mynetworkname")

	networkMethod := suite.TestMAASObject.GetSubObject("networks")
	params := url.Values{"node": []string{"mysystemid"}}
	listNetworkObjects, err := networkMethod.CallGet("", params)
	c.Assert(err, IsNil)

	networkJSONArray, err := listNetworkObjects.GetArray()
	c.Assert(err, IsNil)
	c.Check(networkJSONArray, HasLen, 1)

	listNetworks, err := networkJSONArray[0].GetMAASObject()
	c.Assert(err, IsNil)

	networkName, err := listNetworks.GetField("name")
	c.Assert(err, IsNil)
	c.Check(networkName, Equals, "mynetworkname")
}

func (suite *TestMAASObjectSuite) TestGetNetworksNone(c *C) {
	nodeJSON := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(nodeJSON)

	networkMethod := suite.TestMAASObject.GetSubObject("networks")
	params := url.Values{"node": []string{"mysystemid"}}
	listNetworkObjects, err := networkMethod.CallGet("", params)
	c.Assert(err, IsNil)

	networkJSONArray, err := listNetworkObjects.GetArray()
	c.Assert(err, IsNil)
	c.Check(networkJSONArray, HasLen, 0)
}

func (suite *TestMAASObjectSuite) TestListNodesWithNetworks(c *C) {
	nodeJSON := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(nodeJSON)
	networkJSON := `{"name": "mynetworkname"}`
	suite.TestMAASObject.TestServer.NewNetwork(networkJSON)
	suite.TestMAASObject.TestServer.ConnectNodeToNetworkWithMACAddress("mysystemid", "mynetworkname", "aa:bb:cc:dd:ee:ff")

	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	listNodeObjects, err := nodeListing.CallGet("list", url.Values{})
	c.Assert(err, IsNil)

	listNodes, err := listNodeObjects.GetArray()
	c.Assert(err, IsNil)
	c.Check(listNodes, HasLen, 1)

	node, err := listNodes[0].GetMAASObject()
	c.Assert(err, IsNil)
	systemId, err := node.GetField("system_id")
	c.Assert(err, IsNil)
	c.Check(systemId, Equals, "mysystemid")

	gotResourceURI, err := node.GetField(resourceURI)
	c.Assert(err, IsNil)
	apiVersion := suite.TestMAASObject.TestServer.version
	expectedResourceURI := fmt.Sprintf("/api/%s/nodes/mysystemid/", apiVersion)
	c.Check(gotResourceURI, Equals, expectedResourceURI)

	macAddressSet, err := node.GetMap()["macaddress_set"].GetArray()
	c.Assert(err, IsNil)
	c.Check(macAddressSet, HasLen, 1)

	macAddress, err := macAddressSet[0].GetMap()
	c.Assert(err, IsNil)
	macAddressString, err := macAddress["mac_address"].GetString()
	c.Check(macAddressString, Equals, "aa:bb:cc:dd:ee:ff")

	gotResourceURI, err = macAddress[resourceURI].GetString()
	c.Assert(err, IsNil)
	expectedResourceURI = fmt.Sprintf("/api/%s/nodes/mysystemid/macs/%s/", apiVersion, url.QueryEscape("aa:bb:cc:dd:ee:ff"))
	c.Check(gotResourceURI, Equals, expectedResourceURI)
}

func (suite *TestMAASObjectSuite) TestListNetworkConnectedMACAddresses(c *C) {
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "node_1"}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "node_2"}`)
	suite.TestMAASObject.TestServer.NewNetwork(`{"name": "net_1"}`)
	suite.TestMAASObject.TestServer.NewNetwork(`{"name": "net_2"}`)
	suite.TestMAASObject.TestServer.ConnectNodeToNetworkWithMACAddress("node_2", "net_2", "aa:bb:cc:dd:ee:22")
	suite.TestMAASObject.TestServer.ConnectNodeToNetworkWithMACAddress("node_1", "net_1", "aa:bb:cc:dd:ee:11")
	suite.TestMAASObject.TestServer.ConnectNodeToNetworkWithMACAddress("node_2", "net_1", "aa:bb:cc:dd:ee:21")
	suite.TestMAASObject.TestServer.ConnectNodeToNetworkWithMACAddress("node_1", "net_2", "aa:bb:cc:dd:ee:12")

	nodeListing := suite.TestMAASObject.GetSubObject("networks").GetSubObject("net_1")
	listNodeObjects, err := nodeListing.CallGet("list_connected_macs", url.Values{})
	c.Assert(err, IsNil)

	listNodes, err := listNodeObjects.GetArray()
	c.Assert(err, IsNil)
	c.Check(listNodes, HasLen, 2)

	node, err := listNodes[0].GetMAASObject()
	c.Assert(err, IsNil)
	macAddress, err := node.GetField("mac_address")
	c.Assert(err, IsNil)
	c.Check(macAddress == "aa:bb:cc:dd:ee:11" || macAddress == "aa:bb:cc:dd:ee:21", Equals, true)
	node1_idx := 0
	if macAddress == "aa:bb:cc:dd:ee:21" {
		node1_idx = 1
	}

	node, err = listNodes[node1_idx].GetMAASObject()
	c.Assert(err, IsNil)
	macAddress, err = node.GetField("mac_address")
	c.Assert(err, IsNil)
	c.Check(macAddress, Equals, "aa:bb:cc:dd:ee:11")
	nodeResourceURI, err := node.GetField(resourceURI)
	c.Assert(err, IsNil)
	apiVersion := suite.TestMAASObject.TestServer.version
	expectedResourceURI := fmt.Sprintf("/api/%s/nodes/node_1/macs/%s/", apiVersion, url.QueryEscape("aa:bb:cc:dd:ee:11"))
	c.Check(nodeResourceURI, Equals, expectedResourceURI)

	node, err = listNodes[1-node1_idx].GetMAASObject()
	c.Assert(err, IsNil)
	macAddress, err = node.GetField("mac_address")
	c.Assert(err, IsNil)
	c.Check(macAddress, Equals, "aa:bb:cc:dd:ee:21")
	nodeResourceURI, err = node.GetField(resourceURI)
	c.Assert(err, IsNil)
	expectedResourceURI = fmt.Sprintf("/api/%s/nodes/node_2/macs/%s/", apiVersion, url.QueryEscape("aa:bb:cc:dd:ee:21"))
	c.Check(nodeResourceURI, Equals, expectedResourceURI)
}

func (suite *TestMAASObjectSuite) TestGetVersion(c *C) {
	networkMethod := suite.TestMAASObject.GetSubObject("version")
	params := url.Values{"node": []string{"mysystemid"}}
	versionObject, err := networkMethod.CallGet("", params)
	c.Assert(err, IsNil)

	versionMap, err := versionObject.GetMap()
	c.Assert(err, IsNil)
	jsonArray, ok := versionMap["capabilities"]
	c.Check(ok, Equals, true)
	capArray, err := jsonArray.GetArray()
	for _, capJSONName := range capArray {
		capName, err := capJSONName.GetString()
		c.Assert(err, IsNil)
		c.Check(capName, Equals, "networks-management")
	}
}

const nodeDetailsXML = `<?xml version="1.0" standalone="yes" ?>
<list>
<node id="node2" claimed="true" class="system" handle="DMI:0001">
 <description>Computer</description>
</node>
</list>`

func (suite *TestMAASObjectSuite) TestNodeDetails(c *C) {
	nodeJSON := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(nodeJSON)
	suite.TestMAASObject.TestServer.AddNodeDetails("mysystemid", nodeDetailsXML)

	obj := suite.TestMAASObject.GetSubObject("nodes").GetSubObject("mysystemid")
	uri := obj.URI()
	result, err := obj.client.Get(uri, "details", nil)
	c.Assert(err, IsNil)

	bsonObj := map[string]interface{}{}
	err = bson.Unmarshal(result, &bsonObj)
	c.Assert(err, IsNil)

	_, ok := bsonObj["lldp"]
	c.Check(ok, Equals, true)
	gotXMLText, ok := bsonObj["lshw"]
	c.Check(ok, Equals, true)
	c.Check(string(gotXMLText.([]byte)), Equals, string(nodeDetailsXML))
}
