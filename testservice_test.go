// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	. "launchpad.net/gocheck"
	"net/http"
	"net/url"
)

type GomaasapiTestServerSuite struct {
	server *TestServer
}

var _ = Suite(&GomaasapiTestServerSuite{})

func (suite *GomaasapiTestServerSuite) SetUpTest(c *C) {
	server := NewTestServer()
	suite.server = server
}

func (suite *GomaasapiTestServerSuite) TearDownTest(c *C) {
	suite.server.Close()
}

func (suite *GomaasapiTestServerSuite) TestNewTestServerReturnsTestServer(c *C) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}
	suite.server.serveMux.HandleFunc("/test/", handler)
	resp, err := http.Get(suite.server.Server.URL + "/test/")

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusAccepted)
}

func (suite *GomaasapiTestServerSuite) TestGetResourceURI(c *C) {
	c.Check(getResourceURI("test"), Equals, "/api/1.0/nodes/test/")
}

func (suite *GomaasapiTestServerSuite) TestHandlesNodeListingUnknownPath(c *C) {
	resp, err := http.Get(suite.server.Server.URL + "/api/1.0/nodes/invalid/path/")

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *GomaasapiTestServerSuite) TestNewNode(c *C) {
	input := `{"system_id": "mysystemid"}`

	newNode := suite.server.NewNode(input)

	c.Check(len(suite.server.nodes), Equals, 1)
	c.Check(suite.server.nodes["mysystemid"], DeepEquals, newNode)
}

func (suite *GomaasapiTestServerSuite) TestGetNodeReturnsNodes(c *C) {
	input := `{"system_id": "mysystemid"}`

	newNode := suite.server.NewNode(input)

	nodesMap := suite.server.GetNodes()
	c.Check(len(nodesMap), Equals, 1)
	c.Check(nodesMap["mysystemid"], DeepEquals, newNode)
}

func (suite *GomaasapiTestServerSuite) TestChangeNode(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	suite.server.ChangeNode("mysystemid", "newfield", "newvalue")

	node, _ := suite.server.nodes["mysystemid"]
	mapObj, _ := node.GetMap()
	field, _ := mapObj["newfield"].GetString()
	c.Check(field, Equals, "newvalue")
}

func (suite *GomaasapiTestServerSuite) TestClearClearsData(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	suite.server.addNodeOperation("mysystemid", "start")

	suite.server.Clear()

	c.Check(len(suite.server.nodes), Equals, 0)
	c.Check(len(suite.server.nodeOperations), Equals, 0)
}

func (suite *GomaasapiTestServerSuite) TestAddNodeOperationPopulatesOperations(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)

	suite.server.addNodeOperation("mysystemid", "start")
	suite.server.addNodeOperation("mysystemid", "stop")

	nodeOperations := suite.server.GetNodeOperations()
	operations := nodeOperations["mysystemid"]
	c.Check(operations, DeepEquals, []string{"start", "stop"})
}

func (suite *GomaasapiTestServerSuite) TestNewNodeRequiresJSONString(c *C) {
	input := `invalid:json`
	defer func() {
		recoveredError := recover().(*json.SyntaxError)
		c.Check(recoveredError, NotNil)
		c.Check(recoveredError.Error(), Matches, ".*invalid character.*")
	}()
	suite.server.NewNode(input)
}

func (suite *GomaasapiTestServerSuite) TestNewNodeRequiresSystemIdKey(c *C) {
	input := `{"test": "test"}`
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		c.Check(recoveredError, Matches, ".*does not contain a 'system_id' value.")
	}()
	suite.server.NewNode(input)
}

func (suite *GomaasapiTestServerSuite) TestHandlesNodeRequestNotFound(c *C) {
	resp, err := http.Get(suite.server.Server.URL + "/api/1.0/nodes/test/")

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *GomaasapiTestServerSuite) TestHandlesNodeUnknownOperation(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	respStart, err := http.Post(suite.server.Server.URL+"/api/1.0/nodes/mysystemid/?op=unknown", "", nil)

	c.Check(err, IsNil)
	c.Check(respStart.StatusCode, Equals, http.StatusBadRequest)
}

func (suite *GomaasapiTestServerSuite) TestHandlesNodeDelete(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	req, err := http.NewRequest("DELETE", suite.server.Server.URL+"/api/1.0/nodes/mysystemid/?op=mysystemid", nil)
	client := &http.Client{}
	resp, err := client.Do(req)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	c.Check(len(suite.server.nodes), Equals, 0)
}

// GomaasapiTestMAASObjectSuite valides that the object created by
// TestMAASObject can be used by the gomaasapi library as if it were a real
// MAAS server.
type GomaasapiTestMAASObjectSuite struct {
	TestMAASObject *TestMAASObject
}

var _ = Suite(&GomaasapiTestMAASObjectSuite{})

func (s *GomaasapiTestMAASObjectSuite) SetUpSuite(c *C) {
	s.TestMAASObject = NewTestMAAS()
}

func (s *GomaasapiTestMAASObjectSuite) TearDownSuite(c *C) {
	s.TestMAASObject.Close()
}

func (s *GomaasapiTestMAASObjectSuite) TearDownTest(c *C) {
	s.TestMAASObject.TestServer.Clear()
}

func (suite *GomaasapiTestMAASObjectSuite) TestListNodes(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")

	listNodeObjects, err := nodeListing.CallGet("list", url.Values{})

	c.Check(err, IsNil)
	listNodes, err := listNodeObjects.GetArray()
	c.Check(err, IsNil)
	c.Check(len(listNodes), Equals, 1)
	node, _ := listNodes[0].GetMAASObject()
	systemId, _ := node.GetField("system_id")
	c.Check(systemId, Equals, "mysystemid")
	resourceURI, _ := node.GetField(resource_uri)
	c.Check(resourceURI, Equals, "/api/1.0/nodes/mysystemid/")
}

func (suite *GomaasapiTestMAASObjectSuite) TestListNodesNoNodes(c *C) {
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	listNodeObjects, err := nodeListing.CallGet("list", url.Values{})
	c.Check(err, IsNil)

	listNodes, err := listNodeObjects.GetArray()

	c.Check(err, IsNil)
	c.Check(len(listNodes), Equals, 0)
}

func (suite *GomaasapiTestMAASObjectSuite) TestListNodesSelectedNodes(c *C) {
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

func (suite *GomaasapiTestMAASObjectSuite) TestDeleteNode(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	listNodeObjects, _ := nodeListing.CallGet("list", url.Values{})
	listNodes, _ := listNodeObjects.GetArray()
	node, _ := listNodes[0].GetMAASObject()

	err := node.Delete()

	c.Check(err, IsNil)
	nodeListing = suite.TestMAASObject.GetSubObject("nodes")
	listNodeObjects, _ = nodeListing.CallGet("list", url.Values{})
	listNodes, _ = listNodeObjects.GetArray()
	c.Check(len(listNodes), Equals, 0)
}

func (suite *GomaasapiTestMAASObjectSuite) TestOperationsOnNode(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	listNodeObjects, _ := nodeListing.CallGet("list", url.Values{})
	listNodes, _ := listNodeObjects.GetArray()
	node, _ := listNodes[0].GetMAASObject()
	operations := []string{"start", "stop", "release"}
	for _, operation := range operations {
		_, err := node.CallPost(operation, url.Values{})
		c.Check(err, IsNil)
	}
}

func (suite *GomaasapiTestMAASObjectSuite) TestOperationsOnNodeGetsRecorded(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.TestMAASObject.TestServer.NewNode(input)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	listNodeObjects, _ := nodeListing.CallGet("list", url.Values{})
	listNodes, _ := listNodeObjects.GetArray()
	node, _ := listNodes[0].GetMAASObject()

	_, err := node.CallPost("start", url.Values{})

	c.Check(err, IsNil)
	nodeOperations := suite.TestMAASObject.TestServer.GetNodeOperations()
	operations := nodeOperations["mysystemid"]
	c.Check(operations, DeepEquals, []string{"start"})
}
