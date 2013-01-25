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

func makeFakeModel() maasModel {
	attrs := make(map[string]JSONObject)
	attrs[resource_uri] = jsonString(makeFakeResourceURI())
	return maasModel{jsonMap: jsonMap(attrs)}
}

func (suite *GomaasapiTestSuite) TestImplementsInterfaces(c *C) {
	obj := makeFakeModel()
	_ = JSONObject(obj)
	_ = MAASModel(obj)
}

// maasModels convert only to map or to model.
func (suite *GomaasapiTestSuite) TestConversionsModel(c *C) {
	input := map[string]JSONObject{resource_uri: jsonString("someplace")}
	obj := maasModel{jsonMap: jsonMap(input)}

	mp, err := obj.GetMap()
	c.Check(err, IsNil)
	text, err := mp[resource_uri].GetString()
	c.Check(err, IsNil)
	c.Check(text, Equals, "someplace")

	model, err := obj.GetModel()
	c.Check(err, IsNil)
	_ = model.(maasModel)

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
	obj := maasModel{jsonMap: jsonMap(input)}
	c.Check(obj.URL(), Equals, uri)
}
