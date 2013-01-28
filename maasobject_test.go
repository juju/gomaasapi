// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"fmt"
	. "launchpad.net/gocheck"
	"math/rand"
)

func makeFakeResourceURI() string {
	return "http://example.com/" + fmt.Sprint(rand.Int31())
}

func makeFakeMAASObject() jsonMAASObject {
	attrs := make(map[string]JSONObject)
	attrs[resource_uri] = jsonString(makeFakeResourceURI())
	return jsonMAASObject{jsonMap: jsonMap(attrs)}
}

// jsonMAASObjects convert only to map or to MAASObject.
func (suite *GomaasapiTestSuite) TestConversionsMAASObject(c *C) {
	input := map[string]JSONObject{resource_uri: jsonString("someplace")}
	obj := jsonMAASObject{jsonMap: jsonMap(input)}

	mp, err := obj.GetMap()
	c.Check(err, IsNil)
	text, err := mp[resource_uri].GetString()
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

func (suite *GomaasapiTestSuite) TestURL(c *C) {
	uri := "http://example.com/a/resource"
	input := map[string]JSONObject{resource_uri: jsonString(uri)}
	obj := jsonMAASObject{jsonMap: jsonMap(input)}
	c.Check(obj.URL(), Equals, uri)
}

func (suite *GomaasapiTestSuite) TestGetSubObject(c *C) {
	uri := "http://example.com/a/resource"
	input := map[string]JSONObject{resource_uri: jsonString(uri)}
	obj := jsonMAASObject{jsonMap: jsonMap(input)}
	subName := "/test"
	subObj := obj.GetSubObject(subName)
	c.Check(subObj.URL(), Equals, uri+subName)
}

func (suite *GomaasapiTestSuite) TestGetField(c *C) {
	uri := "http://example.com/a/resource"
	fieldName := "field name"
	fieldValue := "a value"
	input := map[string]JSONObject{
		resource_uri: jsonString(uri), fieldName: jsonString(fieldValue),
	}
	obj := jsonMAASObject{jsonMap: jsonMap(input)}
	value, err := obj.GetField(fieldName)
	c.Check(err, IsNil)
	c.Check(value, Equals, fieldValue)
}
