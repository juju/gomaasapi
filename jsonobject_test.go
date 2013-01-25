// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"launchpad.net/gocheck"
)


// maasify() converts nil.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsNil(c *gocheck.C) {
	c.Check(maasify(nil, nil), gocheck.Equals, nil)
}


// maasify() converts strings.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsString(c *gocheck.C) {
	const text = "Hello"
	c.Check(string(maasify(nil, text).(jsonString)), gocheck.Equals, text)
}


// maasify() converts float64 numbers.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsNumber(c *gocheck.C) {
	const number = 3.1415926535
	c.Check(float64(maasify(nil, number).(jsonFloat64)), gocheck.Equals, number)
}


// maasify() converts array slices.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsArray(c *gocheck.C) {
	original := []interface{}{3.0, 2.0, 1.0}
	output := maasify(nil, original).(jsonArray)
	c.Check(len(output), gocheck.Equals, len(original))
}


// When maasify() converts an array slice, the result contains JSONObjects.
func (suite *GomaasapiTestSuite) TestMaasifyArrayContainsJSONObjects(c *gocheck.C) {
	arr := maasify(nil, []interface{}{9.9}).(jsonArray)
	var entry JSONObject
	entry = arr[0]
	c.Check((float64)(entry.(jsonFloat64)), gocheck.Equals, 9.9)
}


// maasify() converts maps.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsMap(c *gocheck.C) {
	original := map[string]interface{}{"1": "one", "2": "two", "3": "three"}
	output := maasify(nil, original).(jsonMap)
	c.Check(len(output), gocheck.Equals, len(original))
}


// When maasify() converts a map, the result contains JSONObjects.
func (suite *GomaasapiTestSuite) TestMaasifyMapContainsJSONObjects(c *gocheck.C) {
	mp := maasify(nil, map[string]interface{}{"key": "value"}).(jsonMap)
	var entry JSONObject
	entry = mp["key"]
	c.Check((string)(entry.(jsonString)), gocheck.Equals, "value")
}


// maasify() converts MAAS model objects.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsModel(c *gocheck.C) {
	original := map[string]interface{}{
		"resource_uri": "http://example.com/foo",
		"size": "3",
	}
	output := maasify(nil, original).(maasModel)
	c.Check(len(output.jsonMap), gocheck.Equals, len(original))
	c.Check((string)(output.jsonMap["size"].(jsonString)), gocheck.Equals, "3")
}


// maasify() passes its Client to a MAASModel it creates.
func (suite *GomaasapiTestSuite) TestMaasifyPassesClientToModel(c *gocheck.C) {
	client := &genericClient{}
	original := map[string]interface{}{"resource_uri": "http://example.com/foo"}
	output := maasify(client, original).(maasModel)
	c.Check(output.client, gocheck.Equals, client)
}


// maasify() passes its Client into an array of MAASModels it creates.
func (suite *GomaasapiTestSuite) TestMaasifyPassesClientIntoArray(c *gocheck.C) {
	client := &genericClient{}
	obj := map[string]interface{}{"resource_uri": "http://example.com/foo"}
	list := []interface{}{obj}
	output := maasify(client, list).(jsonArray)
	c.Check(output[0].(maasModel).client, gocheck.Equals, client)
}


// maasify() passes its Client into a map of MAASModels it creates.
func (suite *GomaasapiTestSuite) TestMaasifyPassesClientIntoMap(c *gocheck.C) {
	client := &genericClient{}
	obj := map[string]interface{}{"resource_uri": "http://example.com/foo"}
	mp := map[string]interface{}{"key": obj}
	output := maasify(client, mp).(jsonMap)
	c.Check(output["key"].(maasModel).client, gocheck.Equals, client)
}


// maasify() passes its Client all the way down into any MAASModels in the
// object structure it creates.
func (suite *GomaasapiTestSuite) TestMaasifyPassesClientAllTheWay(c *gocheck.C) {
	client := &genericClient{}
	obj := map[string]interface{}{"resource_uri": "http://example.com/foo"}
	mp := map[string]interface{}{"key": obj}
	list := []interface{}{mp}
	output := maasify(client, list).(jsonArray)
	model := output[0].(jsonMap)["key"]
	c.Check(model.(maasModel).client, gocheck.Equals, client)
}


// maasify() converts Booleans.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsBool(c *gocheck.C) {
	c.Check(bool(maasify(nil, true).(jsonBool)), gocheck.Equals, true)
	c.Check(bool(maasify(nil, false).(jsonBool)), gocheck.Equals, false)
}
