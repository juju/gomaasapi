// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"fmt"
	. "launchpad.net/gocheck"
	"math/rand"
	"net/url"
)

func makeFakeResourceURI() string {
	return "http://example.com/" + fmt.Sprint(rand.Int31())
}

func makeFakeMAASObject() jsonMAASObject {
	attrs := make(map[string]JSONObject)
	attrs[resourceURI] = jsonString(makeFakeResourceURI())
	return jsonMAASObject{jsonMap: jsonMap(attrs)}
}

// jsonMAASObjects convert only to map or to MAASObject.
func (suite *GomaasapiTestSuite) TestConversionsMAASObject(c *C) {
	input := map[string]JSONObject{resourceURI: jsonString("someplace")}
	obj := jsonMAASObject{jsonMap: jsonMap(input)}

	mp, err := obj.GetMap()
	c.Check(err, IsNil)
	text, err := mp[resourceURI].GetString()
	c.Check(err, IsNil)
	c.Check(text, Equals, "someplace")

	maasobj, err := obj.GetMAASObject()
	c.Check(err, IsNil)
	_ = maasobj.(jsonMAASObject)

	_, err = obj.GetString()
	c.Check(err, NotNil)
	_, err = obj.GetFloat64()
	c.Check(err, NotNil)
	_, err = obj.GetArray()
	c.Check(err, NotNil)
	_, err = obj.GetBool()
	c.Check(err, NotNil)
}

func (suite *GomaasapiTestSuite) TestNewJSONMAASObjectPanicsIfNoResourceURI(c *C) {
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		c.Check(recoveredError, Matches, ".*no 'resource_uri' key.*")
	}()
	input := map[string]JSONObject{"test": jsonString("test")}
	newJSONMAASObject(jsonMap(input), Client{})
}

func (suite *GomaasapiTestSuite) TestNewJSONMAASObjectPanicsIfResourceURINotString(c *C) {
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		c.Check(recoveredError, Matches, ".*the value of 'resource_uri' is not a string.*")
	}()
	input := map[string]JSONObject{resourceURI: jsonFloat64(77.7)}
	newJSONMAASObject(jsonMap(input), Client{})
}

func (suite *GomaasapiTestSuite) TestNewJSONMAASObjectPanicsIfResourceURINotURL(c *C) {
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		c.Check(recoveredError, Matches, ".*the value of 'resource_uri' is not a valid URL.*")
	}()
	input := map[string]JSONObject{resourceURI: jsonString("")}
	newJSONMAASObject(jsonMap(input), Client{})
}

func (suite *GomaasapiTestSuite) TestNewJSONMAASObjectSetsUpURI(c *C) {
	URI, _ := url.Parse("http://example.com/a/resource")
	input := map[string]JSONObject{resourceURI: jsonString(URI.String())}
	obj := newJSONMAASObject(jsonMap(input), Client{})
	c.Check(obj.uri, DeepEquals, URI)
}

func (suite *GomaasapiTestSuite) TestURL(c *C) {
	baseURL, _ := url.Parse("http://example.com/")
	uri := "http://example.com/a/resource"
	resourceURL, _ := url.Parse(uri)
	input := map[string]JSONObject{resourceURI: jsonString(uri)}
	client := Client{BaseURL: baseURL}
	obj := newJSONMAASObject(jsonMap(input), client)

	URL := obj.URL()

	c.Check(URL, DeepEquals, resourceURL)
}

func (suite *GomaasapiTestSuite) TestGetSubObject(c *C) {
	baseURL, _ := url.Parse("http://example.com/")
	uri := "http://example.com/a/resource/"
	input := map[string]JSONObject{resourceURI: jsonString(uri)}
	client := Client{BaseURL: baseURL}
	obj := newJSONMAASObject(jsonMap(input), client)
	subName := "/test"

	subObj := obj.GetSubObject(subName)
	subURL := subObj.URL()

	// uri ends with a slash and subName starts with one, but the two paths
	// should be concatenated as "http://example.com/a/resource/test/".
	expectedSubURL, _ := url.Parse("http://example.com/a/resource/test/")
	c.Check(subURL, DeepEquals, expectedSubURL)
}

func (suite *GomaasapiTestSuite) TestGetField(c *C) {
	uri := "http://example.com/a/resource"
	fieldName := "field name"
	fieldValue := "a value"
	input := map[string]JSONObject{
		resourceURI: jsonString(uri), fieldName: jsonString(fieldValue),
	}
	obj := jsonMAASObject{jsonMap: jsonMap(input)}
	value, err := obj.GetField(fieldName)
	c.Check(err, IsNil)
	c.Check(value, Equals, fieldValue)
}
