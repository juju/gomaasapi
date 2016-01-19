// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/bson"
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

func (suite *TestServerSuite) TestSetVersionJSON(c *C) {
	capabilities := `{"capabilities": ["networks-management","static-ipaddresses", "devices-management"]}`
	suite.server.SetVersionJSON(capabilities)

	url := fmt.Sprintf("/api/%s/version/", suite.server.version)
	resp, err := http.Get(suite.server.Server.URL + url)
	c.Assert(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	content, err := readAndClose(resp.Body)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, capabilities)
}

func (suite *TestServerSuite) createDevice(c *C, mac, hostname, parent string) string {
	devicesURL := fmt.Sprintf("/api/%s/devices/", suite.server.version) + "?op=new"
	values := url.Values{}
	values.Add("mac_addresses", mac)
	values.Add("hostname", hostname)
	values.Add("parent", parent)
	result := suite.post(c, devicesURL, values)
	resultMap, err := result.GetMap()
	c.Assert(err, IsNil)
	systemId, err := resultMap["system_id"].GetString()
	c.Assert(err, IsNil)
	return systemId
}

func getString(c *C, object map[string]JSONObject, key string) string {
	value, err := object[key].GetString()
	c.Assert(err, IsNil)
	return value
}

func (suite *TestServerSuite) post(c *C, url string, values url.Values) JSONObject {
	resp, err := http.Post(suite.server.Server.URL+url, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	c.Assert(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)
	content, err := readAndClose(resp.Body)
	c.Assert(err, IsNil)
	result, err := Parse(suite.server.client, content)
	c.Assert(err, IsNil)
	return result
}

func (suite *TestServerSuite) get(c *C, url string) JSONObject {
	resp, err := http.Get(suite.server.Server.URL + url)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	content, err := readAndClose(resp.Body)
	c.Assert(err, IsNil)

	result, err := Parse(suite.server.client, content)
	c.Assert(err, IsNil)
	return result
}

func checkDevice(c *C, device map[string]JSONObject, mac, hostname, parent string) {
	macArray, err := device["macaddress_set"].GetArray()
	c.Assert(err, IsNil)
	c.Assert(macArray, HasLen, 1)
	macMap, err := macArray[0].GetMap()
	c.Assert(err, IsNil)

	actualMac := getString(c, macMap, "mac_address")
	c.Assert(actualMac, Equals, mac)

	actualParent := getString(c, device, "parent")
	c.Assert(actualParent, Equals, parent)
	actualHostname := getString(c, device, "hostname")
	c.Assert(actualHostname, Equals, hostname)
}

func (suite *TestServerSuite) TestNewDeviceRequiredParameters(c *C) {
	devicesURL := fmt.Sprintf("/api/%s/devices/", suite.server.version) + "?op=new"
	values := url.Values{}
	values.Add("mac_addresses", "foo")
	values.Add("hostname", "bar")
	post := func(values url.Values) int {
		resp, err := http.Post(suite.server.Server.URL+devicesURL, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
		c.Assert(err, IsNil)
		return resp.StatusCode
	}
	c.Check(post(values), Equals, http.StatusBadRequest)
	values.Del("hostname")
	values.Add("parent", "baz")
	c.Check(post(values), Equals, http.StatusBadRequest)
	values.Del("mac_addresses")
	values.Add("hostname", "bam")
	c.Check(post(values), Equals, http.StatusBadRequest)
}

func (suite *TestServerSuite) TestNewDevice(c *C) {
	devicesURL := fmt.Sprintf("/api/%s/devices/", suite.server.version) + "?op=new"

	values := url.Values{}
	values.Add("mac_addresses", "foo")
	values.Add("hostname", "bar")
	values.Add("parent", "baz")
	result := suite.post(c, devicesURL, values)

	resultMap, err := result.GetMap()
	c.Assert(err, IsNil)

	macArray, err := resultMap["macaddress_set"].GetArray()
	c.Assert(err, IsNil)
	c.Assert(macArray, HasLen, 1)
	macMap, err := macArray[0].GetMap()
	c.Assert(err, IsNil)

	mac := getString(c, macMap, "mac_address")
	c.Assert(mac, Equals, "foo")

	parent := getString(c, resultMap, "parent")
	c.Assert(parent, Equals, "baz")
	hostname := getString(c, resultMap, "hostname")
	c.Assert(hostname, Equals, "bar")

	addresses, err := resultMap["ip_addresses"].GetArray()
	c.Assert(err, IsNil)
	c.Assert(addresses, HasLen, 0)

	systemId := getString(c, resultMap, "system_id")
	resourceURI := getString(c, resultMap, "resource_uri")
	c.Assert(resourceURI, Equals, fmt.Sprintf("/MAAS/api/%v/devices/%v/", suite.server.version, systemId))
}

func (suite *TestServerSuite) TestGetDevice(c *C) {
	systemId := suite.createDevice(c, "foo", "bar", "baz")
	deviceURL := fmt.Sprintf("/api/%v/devices/%v/", suite.server.version, systemId)

	result := suite.get(c, deviceURL)
	resultMap, err := result.GetMap()
	c.Assert(err, IsNil)
	checkDevice(c, resultMap, "foo", "bar", "baz")
	actualId, err := resultMap["system_id"].GetString()
	c.Assert(actualId, Equals, systemId)
}

func (suite *TestServerSuite) TestDevicesList(c *C) {
	firstId := suite.createDevice(c, "foo", "bar", "baz")
	c.Assert(firstId, Not(Equals), "")
	secondId := suite.createDevice(c, "bam", "bing", "bong")
	c.Assert(secondId, Not(Equals), "")

	devicesURL := fmt.Sprintf("/api/%s/devices/", suite.server.version) + "?op=list"
	result := suite.get(c, devicesURL)

	devicesArray, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Assert(devicesArray, HasLen, 2)

	for _, device := range devicesArray {
		deviceMap, err := device.GetMap()
		c.Assert(err, IsNil)
		systemId, err := deviceMap["system_id"].GetString()
		c.Assert(err, IsNil)
		switch systemId {
		case firstId:
			checkDevice(c, deviceMap, "foo", "bar", "baz")
		case secondId:
			checkDevice(c, deviceMap, "bam", "bing", "bong")
		default:
			c.Fatalf("unknown system id %q", systemId)
		}
	}
}

func (suite *TestServerSuite) TestDevicesListMacFiltering(c *C) {
	firstId := suite.createDevice(c, "foo", "bar", "baz")
	c.Assert(firstId, Not(Equals), "")
	secondId := suite.createDevice(c, "bam", "bing", "bong")
	c.Assert(secondId, Not(Equals), "")

	op := fmt.Sprintf("?op=list&mac_address=%v", "foo")
	devicesURL := fmt.Sprintf("/api/%s/devices/", suite.server.version) + op
	result := suite.get(c, devicesURL)

	devicesArray, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Assert(devicesArray, HasLen, 1)
	deviceMap, err := devicesArray[0].GetMap()
	c.Assert(err, IsNil)
	checkDevice(c, deviceMap, "foo", "bar", "baz")
}

func (suite *TestServerSuite) TestDeviceClaimStickyIPRequiresAddress(c *C) {
	systemId := suite.createDevice(c, "foo", "bar", "baz")
	op := "?op=claim_sticky_ip_address"
	deviceURL := fmt.Sprintf("/api/%s/devices/%s/%s", suite.server.version, systemId, op)
	values := url.Values{}
	resp, err := http.Post(suite.server.Server.URL+deviceURL, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusBadRequest)
}

func (suite *TestServerSuite) TestDeviceClaimStickyIP(c *C) {
	systemId := suite.createDevice(c, "foo", "bar", "baz")
	op := "?op=claim_sticky_ip_address"
	deviceURL := fmt.Sprintf("/api/%s/devices/%s/", suite.server.version, systemId)
	values := url.Values{}
	values.Add("requested_address", "127.0.0.1")
	result := suite.post(c, deviceURL+op, values)
	resultMap, err := result.GetMap()
	c.Assert(err, IsNil)

	addresses, err := resultMap["ip_addresses"].GetArray()
	c.Assert(err, IsNil)
	c.Assert(addresses, HasLen, 1)
	address, err := addresses[0].GetString()
	c.Assert(err, IsNil)
	c.Assert(address, Equals, "127.0.0.1")
}

func (suite *TestServerSuite) TestDeleteDevice(c *C) {
	systemId := suite.createDevice(c, "foo", "bar", "baz")
	deviceURL := fmt.Sprintf("/api/%s/devices/%s/", suite.server.version, systemId)
	req, err := http.NewRequest("DELETE", suite.server.Server.URL+deviceURL, nil)
	c.Assert(err, IsNil)
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)

	resp, err = http.Get(suite.server.Server.URL + deviceURL)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
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

func (suite *TestServerSuite) TestHandlesNodegroupsInterfacesListingUnknownNodegroup(c *C) {
	invalidPath := fmt.Sprintf("/api/%s/nodegroups/unknown/interfaces/", suite.server.version)
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

func (suite *TestServerSuite) TestListZonesNotSupported(c *C) {
	// Older versions of MAAS do not support zones. We simulate
	// this behaviour by returning 404 if no zones are defined.
	zonesURL := getZonesEndpoint(suite.server.version)
	resp, err := http.Get(suite.server.Server.URL + zonesURL)

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func defaultSubnet() CreateSubnet {
	var s CreateSubnet
	s.DNSServers = []string{"192.168.1.2"}
	s.Name = "maas-eth0"
	s.Space = "space-0"
	s.GatewayIP = "192.168.1.1"
	s.CIDR = "192.168.1.0/24"
	s.ID = 1
	return s
}

func (suite *TestServerSuite) subnetJSON(subnet CreateSubnet) *bytes.Buffer {
	var out bytes.Buffer
	err := json.NewEncoder(&out).Encode(subnet)
	if err != nil {
		panic(err)
	}
	return &out
}

func (suite *TestServerSuite) subnetURL(ID int) string {
	return suite.subnetsURL() + strconv.Itoa(ID) + "/"
}

func (suite *TestServerSuite) subnetsURL() string {
	return suite.server.Server.URL + getSubnetsEndpoint(suite.server.version)
}

func (suite *TestServerSuite) getSubnets(c *C) []Subnet {
	resp, err := http.Get(suite.subnetsURL())

	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)

	var subnets []Subnet
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&subnets)
	c.Check(err, IsNil)
	return subnets
}

func (suite *TestServerSuite) TestSubnetAdd(c *C) {
	suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))

	subnets := suite.getSubnets(c)
	c.Check(subnets, HasLen, 1)
	s := subnets[0]
	c.Check(s.DNSServers, DeepEquals, []string{"192.168.1.2"})
	c.Check(s.Name, Equals, "maas-eth0")
	c.Check(s.Space, Equals, "space-0")
	c.Check(s.VLAN.ID, Equals, uint(0))
	c.Check(s.CIDR, Equals, "192.168.1.0/24")
}

func (suite *TestServerSuite) TestSubnetGet(c *C) {
	suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))

	subnet2 := defaultSubnet()
	subnet2.Name = "maas-eth1"
	subnet2.CIDR = "192.168.2.0/24"
	suite.server.NewSubnet(suite.subnetJSON(subnet2))

	subnets := suite.getSubnets(c)
	c.Check(subnets, HasLen, 2)
	c.Check(subnets[0].CIDR, Equals, "192.168.1.0/24")
	c.Check(subnets[1].CIDR, Equals, "192.168.2.0/24")
}

func (suite *TestServerSuite) TestSubnetPut(c *C) {
	subnet1 := defaultSubnet()
	suite.server.NewSubnet(suite.subnetJSON(subnet1))

	subnets := suite.getSubnets(c)
	c.Check(subnets, HasLen, 1)
	c.Check(subnets[0].DNSServers, DeepEquals, []string{"192.168.1.2"})

	subnet1.DNSServers = []string{"192.168.1.2", "192.168.1.3"}
	suite.server.UpdateSubnet(suite.subnetJSON(subnet1))

	subnets = suite.getSubnets(c)
	c.Check(subnets, HasLen, 1)
	c.Check(subnets[0].DNSServers, DeepEquals, []string{"192.168.1.2", "192.168.1.3"})
}

func (suite *TestServerSuite) TestSubnetDelete(c *C) {
	suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))

	subnets := suite.getSubnets(c)
	c.Check(subnets, HasLen, 1)
	c.Check(subnets[0].DNSServers, DeepEquals, []string{"192.168.1.2"})

	req, err := http.NewRequest("DELETE", suite.subnetURL(1), nil)
	c.Check(err, IsNil)
	resp, err := http.DefaultClient.Do(req)
	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusOK)

	resp, err = http.Get(suite.subnetsURL())
	c.Check(err, IsNil)
	c.Check(resp.StatusCode, Equals, http.StatusNotFound)
}

func (suite *TestServerSuite) reserveSomeAddresses() map[int]bool {
	reserved := make(map[int]bool)
	rand.Seed(6)

	// Insert some random test data
	for i := 0; i < 200; i++ {
		r := rand.Intn(253) + 1
		_, ok := reserved[r]
		for ok == true {
			r++
			if r == 255 {
				r = 1
			}
			_, ok = reserved[r]
		}
		reserved[r] = true
		addr := fmt.Sprintf("192.168.1.%d", r)
		suite.server.NewIPAddress(addr, "maas-eth0")
	}

	return reserved
}

func (suite *TestServerSuite) TestSubnetReservedIPRanges(c *C) {
	suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))
	reserved := suite.reserveSomeAddresses()

	// Fetch from the server
	reservedIPRangeURL := suite.subnetURL(1) + "?op=reserved_ip_ranges"
	resp, err := http.Get(reservedIPRangeURL)
	c.Check(err, IsNil)

	var reservedFromAPI []AddressRange
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&reservedFromAPI)
	c.Check(err, IsNil)

	// Check that anything in a reserved range was an address we allocated
	// with NewIPAddress
	for _, addressRange := range reservedFromAPI {
		var start, end int
		fmt.Sscanf(addressRange.Start, "192.168.1.%d", &start)
		fmt.Sscanf(addressRange.End, "192.168.1.%d", &end)
		c.Check(addressRange.NumAddresses, Equals, uint(1+end-start))
		c.Check(start <= end, Equals, true)
		c.Check(start < 255, Equals, true)
		c.Check(end < 255, Equals, true)
		for i := start; i <= end; i++ {
			_, ok := reserved[int(i)]
			c.Check(ok, Equals, true)
			delete(reserved, int(i))
		}
	}
	c.Check(reserved, HasLen, 0)
}

func (suite *TestServerSuite) TestSubnetUnreservedIPRanges(c *C) {
	suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))
	reserved := suite.reserveSomeAddresses()
	unreserved := make(map[int]bool)

	// Fetch from the server
	reservedIPRangeURL := suite.subnetURL(1) + "?op=unreserved_ip_ranges"
	resp, err := http.Get(reservedIPRangeURL)
	c.Check(err, IsNil)

	var unreservedFromAPI []AddressRange
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&unreservedFromAPI)
	c.Check(err, IsNil)

	// Check that anything in an unreserved range wasn't an address we allocated
	// with NewIPAddress
	for _, addressRange := range unreservedFromAPI {
		var start, end int
		fmt.Sscanf(addressRange.Start, "192.168.1.%d", &start)
		fmt.Sscanf(addressRange.End, "192.168.1.%d", &end)
		c.Check(addressRange.NumAddresses, Equals, uint(1+end-start))
		c.Check(start <= end, Equals, true)
		c.Check(start < 255, Equals, true)
		c.Check(end < 255, Equals, true)
		for i := start; i <= end; i++ {
			_, ok := reserved[int(i)]
			c.Check(ok, Equals, false)
			unreserved[int(i)] = true
		}
	}
	for i := 1; i < 255; i++ {
		_, r := reserved[i]
		_, u := unreserved[i]
		if (r || u) == false {
			fmt.Println(i, r, u)
		}
		c.Check(r || u, Equals, true)
	}
	c.Check(len(reserved)+len(unreserved), Equals, 254)
}

func (suite *TestServerSuite) TestSubnetReserveRange(c *C) {
	suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))
	suite.server.NewIPAddress("192.168.1.10", "maas-eth0")

	var ar AddressRange
	ar.Start = "192.168.1.100"
	ar.End = "192.168.1.200"
	ar.Purpose = []string{"dynamic"}

	suite.server.AddFixedAddressRange(1, ar)

	// Fetch from the server
	reservedIPRangeURL := suite.subnetURL(1) + "?op=reserved_ip_ranges"
	resp, err := http.Get(reservedIPRangeURL)
	c.Check(err, IsNil)

	var reservedFromAPI []AddressRange
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&reservedFromAPI)
	c.Check(err, IsNil)

	// Check that the address ranges we got back were as expected
	addressRange := reservedFromAPI[0]
	c.Check(addressRange.Start, Equals, "192.168.1.10")
	c.Check(addressRange.End, Equals, "192.168.1.10")
	c.Check(addressRange.NumAddresses, Equals, uint(1))
	c.Check(addressRange.Purpose[0], Equals, "assigned-ip")
	c.Check(addressRange.Purpose, HasLen, 1)

	addressRange = reservedFromAPI[1]
	c.Check(addressRange.Start, Equals, "192.168.1.100")
	c.Check(addressRange.End, Equals, "192.168.1.200")
	c.Check(addressRange.NumAddresses, Equals, uint(101))
	c.Check(addressRange.Purpose[0], Equals, "dynamic")
	c.Check(addressRange.Purpose, HasLen, 1)
}

func (suite *TestServerSuite) getSubnetStats(c *C, subnetID int) SubnetStats {
	URL := suite.subnetURL(1) + "?op=statistics"
	resp, err := http.Get(URL)
	c.Check(err, IsNil)

	var s SubnetStats
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&s)
	c.Check(err, IsNil)
	return s
}

func (suite *TestServerSuite) TestSubnetStats(c *C) {
	suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))

	stats := suite.getSubnetStats(c, 1)
	// There are 254 usable addresses in a class C subnet, so these
	// stats are fixed
	expected := SubnetStats{
		NumAvailable:     254,
		LargestAvailable: 254,
		NumUnavailable:   0,
		TotalAddresses:   254,
		Usage:            0,
		UsageString:      "0.0%",
		Ranges:           nil,
	}
	c.Check(stats, DeepEquals, expected)

	suite.reserveSomeAddresses()
	stats = suite.getSubnetStats(c, 1)
	// We have reserved 200 addresses so parts of these
	// stats are fixed.
	expected = SubnetStats{
		NumAvailable:   54,
		NumUnavailable: 200,
		TotalAddresses: 254,
		Usage:          0.787401556968689,
		UsageString:    "78.7%",
		Ranges:         nil,
	}

	reserved := suite.server.subnetUnreservedIPRanges(suite.server.subnets[1])
	var largestAvailable uint
	for _, addressRange := range reserved {
		if addressRange.NumAddresses > largestAvailable {
			largestAvailable = addressRange.NumAddresses
		}
	}

	expected.LargestAvailable = largestAvailable
	c.Check(stats, DeepEquals, expected)
}

func (suite *TestServerSuite) TestSubnetsInNodes(c *C) {
	// Create a subnet
	subnet := suite.server.NewSubnet(suite.subnetJSON(defaultSubnet()))

	// Create a node
	var node Node
	node.SystemID = "node-89d832ca-8877-11e5-b5a5-00163e86022b"
	suite.server.NewNode(fmt.Sprintf(`{"system_id": "%s"}`, "node-89d832ca-8877-11e5-b5a5-00163e86022b"))

	// Put the node in the subnet
	var nni NodeNetworkInterface
	nni.Name = "eth0"
	nni.Links = append(nni.Links, NetworkLink{uint(1), "auto", subnet})
	suite.server.SetNodeNetworkLink(node.SystemID, nni)

	// Fetch the node details
	URL := suite.server.Server.URL + getNodesEndpoint(suite.server.version) + node.SystemID + "/"
	resp, err := http.Get(URL)
	c.Check(err, IsNil)

	var n Node
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&n)
	c.Check(err, IsNil)
	c.Check(n.SystemID, Equals, node.SystemID)
	c.Check(n.Interfaces, HasLen, 1)
	i := n.Interfaces[0]
	c.Check(i.Name, Equals, "eth0")
	c.Check(i.Links, HasLen, 1)
	c.Check(i.Links[0].ID, Equals, uint(1))
	c.Check(i.Links[0].Subnet.Name, Equals, "maas-eth0")
}

type IPSuite struct {
}

var _ = Suite(&IPSuite{})

func (suite *IPSuite) TestIPFromNetIP(c *C) {
	ip := IPFromNetIP(net.ParseIP("1.2.3.4"))
	c.Check(ip.String(), Equals, "1.2.3.4")
}

func (suite *IPSuite) TestIPUInt64(c *C) {
	ip := IPFromNetIP(net.ParseIP("1.2.3.4"))
	v := ip.UInt64()
	c.Check(v, Equals, uint64(0x01020304))
}

func (suite *IPSuite) TestIPSetUInt64(c *C) {
	var ip IP
	ip.SetUInt64(0x01020304)
	c.Check(ip.String(), Equals, "1.2.3.4")
}

// TestMAASObjectSuite validates that the object created by
// NewTestMAAS can be used by the gomaasapi library as if it were a real
// MAAS server.
type TestMAASObjectSuite struct {
	TestMAASObject *TestMAASObject
}

var _ = Suite(&TestMAASObjectSuite{})

func (suite *TestMAASObjectSuite) SetUpSuite(c *C) {
	suite.TestMAASObject = NewTestMAAS("1.0")
}

func (suite *TestMAASObjectSuite) TearDownSuite(c *C) {
	suite.TestMAASObject.Close()
}

func (suite *TestMAASObjectSuite) TearDownTest(c *C) {
	suite.TestMAASObject.TestServer.Clear()
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
	networkJSON := `{"name": "mynetworkname", "ip": "0.1.2.0", "netmask": "255.255.255.0"}`
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
	ip, err := listNetworks.GetField("ip")
	c.Assert(err, IsNil)
	netmask, err := listNetworks.GetField("netmask")
	c.Assert(err, IsNil)
	c.Check(networkName, Equals, "mynetworkname")
	c.Check(ip, Equals, "0.1.2.0")
	c.Check(netmask, Equals, "255.255.255.0")
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
	networkJSON := `{"name": "mynetworkname", "ip": "0.1.2.0", "netmask": "255.255.255.0"}`
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
	suite.TestMAASObject.TestServer.NewNetwork(
		`{"name": "net_1", "ip": "0.1.2.0", "netmask": "255.255.255.0"}`,
	)
	suite.TestMAASObject.TestServer.NewNetwork(
		`{"name": "net_2", "ip": "0.2.2.0", "netmask": "255.255.255.0"}`,
	)
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
		switch capName {
		case "networks-management":
		case "static-ipaddresses":
		case "devices-management":
		case "network-deployment-ubuntu":
		default:
			c.Fatalf("unknown capability %q", capName)
		}
	}
}

func (suite *TestMAASObjectSuite) assertIPAmong(c *C, jsonObjIP JSONObject, expectIPs ...string) {
	apiVersion := suite.TestMAASObject.TestServer.version
	expectedURI := getIPAddressesEndpoint(apiVersion)

	maasObj, err := jsonObjIP.GetMAASObject()
	c.Assert(err, IsNil)
	attrs := maasObj.GetMap()
	uri, err := attrs["resource_uri"].GetString()
	c.Assert(err, IsNil)
	c.Assert(uri, Equals, expectedURI)
	allocType, err := attrs["alloc_type"].GetFloat64()
	c.Assert(err, IsNil)
	c.Assert(allocType, Equals, 4.0)
	created, err := attrs["created"].GetString()
	c.Assert(err, IsNil)
	c.Assert(created, Not(Equals), "")
	ip, err := attrs["ip"].GetString()
	c.Assert(err, IsNil)
	if !contains(expectIPs, ip) {
		c.Fatalf("expected IP in %v, got %q", expectIPs, ip)
	}
}

func (suite *TestMAASObjectSuite) TestListIPAddresses(c *C) {
	ipAddresses := suite.TestMAASObject.GetSubObject("ipaddresses")

	// First try without any networks and IPs.
	listIPObjects, err := ipAddresses.CallGet("", url.Values{})
	c.Assert(err, IsNil)
	items, err := listIPObjects.GetArray()
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 0)

	// Add two networks and some addresses to each one.
	suite.TestMAASObject.TestServer.NewNetwork(
		`{"name": "net_1", "ip": "0.1.2.0", "netmask": "255.255.255.0"}`,
	)
	suite.TestMAASObject.TestServer.NewNetwork(
		`{"name": "net_2", "ip": "0.2.2.0", "netmask": "255.255.255.0"}`,
	)
	suite.TestMAASObject.TestServer.NewIPAddress("0.1.2.3", "net_1")
	suite.TestMAASObject.TestServer.NewIPAddress("0.1.2.4", "net_1")
	suite.TestMAASObject.TestServer.NewIPAddress("0.1.2.5", "net_1")
	suite.TestMAASObject.TestServer.NewIPAddress("0.2.2.3", "net_2")
	suite.TestMAASObject.TestServer.NewIPAddress("0.2.2.4", "net_2")

	// List all addresses and verify the needed response fields are set.
	listIPObjects, err = ipAddresses.CallGet("", url.Values{})
	c.Assert(err, IsNil)
	items, err = listIPObjects.GetArray()
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 5)

	for _, ipObj := range items {
		suite.assertIPAmong(
			c, ipObj,
			"0.1.2.3", "0.1.2.4", "0.1.2.5", "0.2.2.3", "0.2.2.4",
		)
	}

	// Remove all net_1 IPs.
	removed := suite.TestMAASObject.TestServer.RemoveIPAddress("0.1.2.3")
	c.Assert(removed, Equals, true)
	removed = suite.TestMAASObject.TestServer.RemoveIPAddress("0.1.2.4")
	c.Assert(removed, Equals, true)
	removed = suite.TestMAASObject.TestServer.RemoveIPAddress("0.1.2.5")
	c.Assert(removed, Equals, true)
	// Remove the last IP twice, should be OK and return false.
	removed = suite.TestMAASObject.TestServer.RemoveIPAddress("0.1.2.5")
	c.Assert(removed, Equals, false)

	// List again.
	listIPObjects, err = ipAddresses.CallGet("", url.Values{})
	c.Assert(err, IsNil)
	items, err = listIPObjects.GetArray()
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 2)
	for _, ipObj := range items {
		suite.assertIPAmong(
			c, ipObj,
			"0.2.2.3", "0.2.2.4",
		)
	}
}

func (suite *TestMAASObjectSuite) TestReserveIPAddress(c *C) {
	suite.TestMAASObject.TestServer.NewNetwork(
		`{"name": "net_1", "ip": "0.1.2.0", "netmask": "255.255.255.0"}`,
	)
	ipAddresses := suite.TestMAASObject.GetSubObject("ipaddresses")
	// First try "reserve" with requested_address set.
	params := url.Values{"network": []string{"0.1.2.0/24"}, "requested_address": []string{"0.1.2.42"}}
	res, err := ipAddresses.CallPost("reserve", params)
	c.Assert(err, IsNil)
	suite.assertIPAmong(c, res, "0.1.2.42")

	// Now try "reserve" without requested_address.
	delete(params, "requested_address")
	res, err = ipAddresses.CallPost("reserve", params)
	c.Assert(err, IsNil)
	suite.assertIPAmong(c, res, "0.1.2.2")
}

func (suite *TestMAASObjectSuite) TestReleaseIPAddress(c *C) {
	suite.TestMAASObject.TestServer.NewNetwork(
		`{"name": "net_1", "ip": "0.1.2.0", "netmask": "255.255.255.0"}`,
	)
	suite.TestMAASObject.TestServer.NewIPAddress("0.1.2.3", "net_1")
	ipAddresses := suite.TestMAASObject.GetSubObject("ipaddresses")

	// Try with non-existing address - should return 404.
	params := url.Values{"ip": []string{"0.2.2.1"}}
	_, err := ipAddresses.CallPost("release", params)
	c.Assert(err, ErrorMatches, `(\n|.)*404 Not Found(\n|.)*`)

	// Now with existing one - all OK.
	params = url.Values{"ip": []string{"0.1.2.3"}}
	_, err = ipAddresses.CallPost("release", params)
	c.Assert(err, IsNil)

	// Ensure it got removed.
	c.Assert(suite.TestMAASObject.TestServer.ipAddressesPerNetwork["net_1"], HasLen, 0)

	// Try again, should return 404.
	_, err = ipAddresses.CallPost("release", params)
	c.Assert(err, ErrorMatches, `(\n|.)*404 Not Found(\n|.)*`)
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

func (suite *TestMAASObjectSuite) TestListNodegroups(c *C) {
	suite.TestMAASObject.TestServer.AddBootImage("uuid-0", `{"architecture": "arm64", "release": "trusty"}`)
	suite.TestMAASObject.TestServer.AddBootImage("uuid-1", `{"architecture": "amd64", "release": "precise"}`)

	nodegroupListing := suite.TestMAASObject.GetSubObject("nodegroups")
	result, err := nodegroupListing.CallGet("list", nil)
	c.Assert(err, IsNil)

	nodegroups, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Check(nodegroups, HasLen, 2)

	for _, obj := range nodegroups {
		nodegroup, err := obj.GetMAASObject()
		c.Assert(err, IsNil)
		uuid, err := nodegroup.GetField("uuid")
		c.Assert(err, IsNil)

		nodegroupResourceURI, err := nodegroup.GetField(resourceURI)
		c.Assert(err, IsNil)
		apiVersion := suite.TestMAASObject.TestServer.version
		expectedResourceURI := fmt.Sprintf("/api/%s/nodegroups/%s/", apiVersion, uuid)
		c.Check(nodegroupResourceURI, Equals, expectedResourceURI)
	}
}

func (suite *TestMAASObjectSuite) TestListNodegroupsEmptyList(c *C) {
	nodegroupListing := suite.TestMAASObject.GetSubObject("nodegroups")
	result, err := nodegroupListing.CallGet("list", nil)
	c.Assert(err, IsNil)

	nodegroups, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Check(nodegroups, HasLen, 0)
}

func (suite *TestMAASObjectSuite) TestListNodegroupInterfaces(c *C) {
	suite.TestMAASObject.TestServer.AddBootImage("uuid-0", `{"architecture": "arm64", "release": "trusty"}`)
	jsonText := `{
            "ip_range_high": "172.16.0.128",
            "ip_range_low": "172.16.0.2",
            "broadcast_ip": "172.16.0.255",
            "static_ip_range_low": "172.16.0.129",
            "name": "eth0",
            "ip": "172.16.0.2",
            "subnet_mask": "255.255.255.0",
            "management": 2,
            "static_ip_range_high": "172.16.0.255",
            "interface": "eth0"
        }`

	suite.TestMAASObject.TestServer.NewNodegroupInterface("uuid-0", jsonText)
	nodegroupsInterfacesListing := suite.TestMAASObject.GetSubObject("nodegroups").GetSubObject("uuid-0").GetSubObject("interfaces")
	result, err := nodegroupsInterfacesListing.CallGet("list", nil)
	c.Assert(err, IsNil)

	nodegroupsInterfaces, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Check(nodegroupsInterfaces, HasLen, 1)

	nodegroupsInterface, err := nodegroupsInterfaces[0].GetMap()
	c.Assert(err, IsNil)

	checkMember := func(member, expectedValue string) {
		value, err := nodegroupsInterface[member].GetString()
		c.Assert(err, IsNil)
		c.Assert(value, Equals, expectedValue)
	}
	checkMember("ip_range_high", "172.16.0.128")
	checkMember("ip_range_low", "172.16.0.2")
	checkMember("broadcast_ip", "172.16.0.255")
	checkMember("static_ip_range_low", "172.16.0.129")
	checkMember("static_ip_range_high", "172.16.0.255")
	checkMember("name", "eth0")
	checkMember("ip", "172.16.0.2")
	checkMember("subnet_mask", "255.255.255.0")
	checkMember("interface", "eth0")

	value, err := nodegroupsInterface["management"].GetFloat64()
	c.Assert(err, IsNil)
	c.Assert(value, Equals, 2.0)
}

func (suite *TestMAASObjectSuite) TestListNodegroupsInterfacesEmptyList(c *C) {
	suite.TestMAASObject.TestServer.AddBootImage("uuid-0", `{"architecture": "arm64", "release": "trusty"}`)
	nodegroupsInterfacesListing := suite.TestMAASObject.GetSubObject("nodegroups").GetSubObject("uuid-0").GetSubObject("interfaces")
	result, err := nodegroupsInterfacesListing.CallGet("list", nil)
	c.Assert(err, IsNil)

	interfaces, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Check(interfaces, HasLen, 0)
}

func (suite *TestMAASObjectSuite) TestListBootImages(c *C) {
	suite.TestMAASObject.TestServer.AddBootImage("uuid-0", `{"architecture": "arm64", "release": "trusty"}`)
	suite.TestMAASObject.TestServer.AddBootImage("uuid-1", `{"architecture": "amd64", "release": "precise"}`)
	suite.TestMAASObject.TestServer.AddBootImage("uuid-1", `{"architecture": "ppc64el", "release": "precise"}`)

	bootImageListing := suite.TestMAASObject.GetSubObject("nodegroups").GetSubObject("uuid-1").GetSubObject("boot-images")
	result, err := bootImageListing.CallGet("", nil)
	c.Assert(err, IsNil)

	bootImageObjects, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Check(bootImageObjects, HasLen, 2)

	expectedBootImages := []string{"amd64.precise", "ppc64el.precise"}
	bootImages := make([]string, len(bootImageObjects))
	for i, obj := range bootImageObjects {
		bootimage, err := obj.GetMap()
		c.Assert(err, IsNil)
		architecture, err := bootimage["architecture"].GetString()
		c.Assert(err, IsNil)
		release, err := bootimage["release"].GetString()
		c.Assert(err, IsNil)
		bootImages[i] = fmt.Sprintf("%s.%s", architecture, release)
	}
	sort.Strings(bootImages)
	c.Assert(bootImages, DeepEquals, expectedBootImages)
}

func (suite *TestMAASObjectSuite) TestListZones(c *C) {
	expected := map[string]string{
		"zone0": "zone0 is very nice",
		"zone1": "zone1 is much nicer than zone0",
	}
	for name, desc := range expected {
		suite.TestMAASObject.TestServer.AddZone(name, desc)
	}

	result, err := suite.TestMAASObject.GetSubObject("zones").CallGet("", nil)
	c.Assert(err, IsNil)
	c.Assert(result, NotNil)

	list, err := result.GetArray()
	c.Assert(err, IsNil)
	c.Assert(list, HasLen, len(expected))

	m := make(map[string]string)
	for _, item := range list {
		itemMap, err := item.GetMap()
		c.Assert(err, IsNil)
		name, err := itemMap["name"].GetString()
		c.Assert(err, IsNil)
		desc, err := itemMap["description"].GetString()
		c.Assert(err, IsNil)
		m[name] = desc
	}
	c.Assert(m, DeepEquals, expected)
}

func (suite *TestMAASObjectSuite) TestAcquireNodeZone(c *C) {
	suite.TestMAASObject.TestServer.AddZone("z0", "rox")
	suite.TestMAASObject.TestServer.AddZone("z1", "sux")
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n0", "zone": "z0"}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n1", "zone": "z1"}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n2", "zone": "z1"}`)
	nodesObj := suite.TestMAASObject.GetSubObject("nodes")

	acquire := func(zone string) (string, string, error) {
		var params url.Values
		if zone != "" {
			params = url.Values{"zone": []string{zone}}
		}
		jsonResponse, err := nodesObj.CallPost("acquire", params)
		if err != nil {
			return "", "", err
		}
		acquiredNode, err := jsonResponse.GetMAASObject()
		c.Assert(err, IsNil)
		systemId, err := acquiredNode.GetField("system_id")
		c.Assert(err, IsNil)
		assignedZone, err := acquiredNode.GetField("zone")
		c.Assert(err, IsNil)
		if zone != "" {
			c.Assert(assignedZone, Equals, zone)
		}
		return systemId, assignedZone, nil
	}

	id, _, err := acquire("z0")
	c.Assert(err, IsNil)
	c.Assert(id, Equals, "n0")
	id, _, err = acquire("z0")
	c.Assert(err.(ServerError).StatusCode, Equals, http.StatusConflict)

	id, zone, err := acquire("")
	c.Assert(err, IsNil)
	c.Assert(id, Not(Equals), "n0")
	c.Assert(zone, Equals, "z1")
}

func (suite *TestMAASObjectSuite) TestAcquireFilterMemory(c *C) {
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n0", "memory": 1024}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n1", "memory": 2048}`)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	jsonResponse, err := nodeListing.CallPost("acquire", url.Values{"mem": []string{"2048"}})
	c.Assert(err, IsNil)
	acquiredNode, err := jsonResponse.GetMAASObject()
	c.Assert(err, IsNil)
	mem, err := acquiredNode.GetMap()["memory"].GetFloat64()
	c.Assert(err, IsNil)
	c.Assert(mem, Equals, float64(2048))
}

func (suite *TestMAASObjectSuite) TestAcquireFilterCpuCores(c *C) {
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n0", "cpu_count": 1}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n1", "cpu_count": 2}`)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	jsonResponse, err := nodeListing.CallPost("acquire", url.Values{"cpu-cores": []string{"2"}})
	c.Assert(err, IsNil)
	acquiredNode, err := jsonResponse.GetMAASObject()
	c.Assert(err, IsNil)
	cpucount, err := acquiredNode.GetMap()["cpu_count"].GetFloat64()
	c.Assert(err, IsNil)
	c.Assert(cpucount, Equals, float64(2))
}

func (suite *TestMAASObjectSuite) TestAcquireFilterArch(c *C) {
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n0", "architecture": "amd64"}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n1", "architecture": "arm/generic"}`)
	nodeListing := suite.TestMAASObject.GetSubObject("nodes")
	jsonResponse, err := nodeListing.CallPost("acquire", url.Values{"arch": []string{"arm"}})
	c.Assert(err, IsNil)
	acquiredNode, err := jsonResponse.GetMAASObject()
	c.Assert(err, IsNil)
	arch, _ := acquiredNode.GetField("architecture")
	c.Assert(arch, Equals, "arm/generic")
}

func (suite *TestMAASObjectSuite) TestDeploymentStatus(c *C) {
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n0", "status": "6"}`)
	suite.TestMAASObject.TestServer.NewNode(`{"system_id": "n1", "status": "1"}`)
	nodes := suite.TestMAASObject.GetSubObject("nodes")
	jsonResponse, err := nodes.CallGet("deployment_status", url.Values{"nodes": []string{"n0", "n1"}})
	c.Assert(err, IsNil)
	deploymentStatus, err := jsonResponse.GetMap()
	c.Assert(err, IsNil)
	c.Assert(deploymentStatus, HasLen, 2)
	expectedStatus := map[string]string{
		"n0": "Deployed", "n1": "Not in Deployment",
	}
	for systemId, status := range expectedStatus {
		nodeStatus, err := deploymentStatus[systemId].GetString()
		c.Assert(err, IsNil)
		c.Assert(nodeStatus, Equals, status)
	}
}
