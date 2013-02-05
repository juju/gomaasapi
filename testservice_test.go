// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"
	. "launchpad.net/gocheck"
	"net/http"
	"net/url"
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
	c.Check(getNodeURI("version", "test"), Equals, "/api/version/nodes/test/")
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
	mapObj, _ := node.GetMap()
	field, _ := mapObj["newfield"].GetString()
	c.Check(field, Equals, "newvalue")
}

func (suite *TestServerSuite) TestClearClearsData(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)
	suite.server.addNodeOperation("mysystemid", "start")

	suite.server.Clear()

	c.Check(len(suite.server.nodes), Equals, 0)
	c.Check(len(suite.server.nodeOperations), Equals, 0)
}

func (suite *TestServerSuite) TestAddNodeOperationPopulatesOperations(c *C) {
	input := `{"system_id": "mysystemid"}`
	suite.server.NewNode(input)

	suite.server.addNodeOperation("mysystemid", "start")
	suite.server.addNodeOperation("mysystemid", "stop")

	nodeOperations := suite.server.NodeOperations()
	operations := nodeOperations["mysystemid"]
	c.Check(operations, DeepEquals, []string{"start", "stop"})
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
	client := &http.Client{}
	resp, err := client.Do(req)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	c.Check(len(suite.server.nodes), Equals, 0)
}

// TestMAASObjectSuite validates that the object created by
// TestMAASObject can be used by the gomaasapi library as if it were a real
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
	c.Check(err, IsNil)
	c.Check(len(listNodes), Equals, 1)
	node, _ := listNodes[0].GetMAASObject()
	systemId, _ := node.GetField("system_id")
	c.Check(systemId, Equals, "mysystemid")
	resourceURI, _ := node.GetField(resourceURI)
	expectedResourceURI := fmt.Sprintf("/api/%s/nodes/mysystemid/", suite.TestMAASObject.TestServer.version)
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
