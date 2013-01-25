// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"fmt"
	"launchpad.net/gocheck"
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

func (suite *GomaasapiTestSuite) TestImplementsInterfaces(c *gocheck.C) {
	obj := makeFakeModel()
	_ = JSONObject(obj)
	_ = MAASModel(obj)
}

// maasModels convert only to map or to model.
func (suite *GomaasapiTestSuite) TestConversionsModel(c *gocheck.C) {
	input := map[string]JSONObject{resource_uri: jsonString("someplace")}
	obj := maasModel{jsonMap: jsonMap(input)}

	mp, err := obj.GetMap()
	c.Check(err, gocheck.IsNil)
	text, err := mp[resource_uri].GetString()
	c.Check(err, gocheck.IsNil)
	c.Check(text, gocheck.Equals, "someplace")

	model, err := obj.GetModel()
	c.Check(err, gocheck.IsNil)
	_ = model.(maasModel)

	_, err = obj.GetString()
	c.Check(err, gocheck.NotNil)
	_, err = obj.GetFloat64()
	c.Check(err, gocheck.NotNil)
	_, err = obj.GetArray()
	c.Check(err, gocheck.NotNil)
	_, err = obj.GetBool()
	c.Check(err, gocheck.NotNil)
}

func (suite *GomaasapiTestSuite) TestURL(c *gocheck.C) {
	uri := "http://example.com/a/resource"
	input := map[string]JSONObject{resource_uri: jsonString(uri)}
	obj := maasModel{jsonMap: jsonMap(input)}
	c.Check(obj.URL(), gocheck.Equals, uri)
}
