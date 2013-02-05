// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	. "launchpad.net/gocheck"
)

type JSONObjectSuite struct {
}

var _ = Suite(&JSONObjectSuite{})

// maasify() converts nil.
func (suite *JSONObjectSuite) TestMaasifyConvertsNil(c *C) {
	c.Check(maasify(Client{}, nil), Equals, nil)
}

// maasify() converts strings.
func (suite *JSONObjectSuite) TestMaasifyConvertsString(c *C) {
	const text = "Hello"
	c.Check(string(maasify(Client{}, text).(jsonString)), Equals, text)
}

// maasify() converts float64 numbers.
func (suite *JSONObjectSuite) TestMaasifyConvertsNumber(c *C) {
	const number = 3.1415926535
	c.Check(float64(maasify(Client{}, number).(jsonFloat64)), Equals, number)
}

// maasify() converts array slices.
func (suite *JSONObjectSuite) TestMaasifyConvertsArray(c *C) {
	original := []interface{}{3.0, 2.0, 1.0}
	output := maasify(Client{}, original).(jsonArray)
	c.Check(len(output), Equals, len(original))
}

// When maasify() converts an array slice, the result contains JSONObjects.
func (suite *JSONObjectSuite) TestMaasifyArrayContainsJSONObjects(c *C) {
	arr := maasify(Client{}, []interface{}{9.9}).(jsonArray)
	var entry JSONObject
	entry = arr[0]
	c.Check((float64)(entry.(jsonFloat64)), Equals, 9.9)
}

// maasify() converts maps.
func (suite *JSONObjectSuite) TestMaasifyConvertsMap(c *C) {
	original := map[string]interface{}{"1": "one", "2": "two", "3": "three"}
	output := maasify(Client{}, original).(jsonMap)
	c.Check(len(output), Equals, len(original))
}

// When maasify() converts a map, the result contains JSONObjects.
func (suite *JSONObjectSuite) TestMaasifyMapContainsJSONObjects(c *C) {
	mp := maasify(Client{}, map[string]interface{}{"key": "value"}).(jsonMap)
	var entry JSONObject
	entry = mp["key"]
	c.Check((string)(entry.(jsonString)), Equals, "value")
}

// maasify() converts MAAS objects.
func (suite *JSONObjectSuite) TestMaasifyConvertsMAASObject(c *C) {
	original := map[string]interface{}{
		"resource_uri": "http://example.com/foo",
		"size":         "3",
	}
	output := maasify(Client{}, original).(jsonMAASObject)
	c.Check(len(output.jsonMap), Equals, len(original))
	c.Check((string)(output.jsonMap["size"].(jsonString)), Equals, "3")
}

// maasify() passes its client to a MAASObject it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesInfoToMAASObject(c *C) {
	client := Client{}
	original := map[string]interface{}{"resource_uri": "/foo"}
	output := maasify(client, original).(jsonMAASObject)
	c.Check(output.client, Equals, client)
}

// maasify() passes its client into an array of MAASObjects it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesInfoIntoArray(c *C) {
	client := Client{}
	obj := map[string]interface{}{"resource_uri": "/foo"}
	list := []interface{}{obj}
	output := maasify(client, list).(jsonArray)
	c.Check(output[0].(jsonMAASObject).client, Equals, client)
}

// maasify() passes its client into a map of MAASObjects it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesInfoIntoMap(c *C) {
	client := Client{}
	obj := map[string]interface{}{"resource_uri": "/foo"}
	mp := map[string]interface{}{"key": obj}
	output := maasify(client, mp).(jsonMap)
	c.Check(output["key"].(jsonMAASObject).client, Equals, client)
}

// maasify() passes its client all the way down into any MAASObjects in the
// object structure it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesInfoAllTheWay(c *C) {
	client := Client{}
	obj := map[string]interface{}{"resource_uri": "/foo"}
	mp := map[string]interface{}{"key": obj}
	list := []interface{}{mp}
	output := maasify(client, list).(jsonArray)
	maasobj := output[0].(jsonMap)["key"]
	c.Check(maasobj.(jsonMAASObject).client, Equals, client)
}

// maasify() converts Booleans.
func (suite *JSONObjectSuite) TestMaasifyConvertsBool(c *C) {
	c.Check(bool(maasify(Client{}, true).(jsonBool)), Equals, true)
	c.Check(bool(maasify(Client{}, false).(jsonBool)), Equals, false)
}

// Parse takes you from a JSON blob to a JSONObject.
func (suite *JSONObjectSuite) TestParseMaasifiesJSONBlob(c *C) {
	blob := []byte("[12]")
	obj, err := Parse(Client{}, blob)
	c.Check(err, IsNil)
	c.Check(float64(obj.(jsonArray)[0].(jsonFloat64)), Equals, 12.0)
}

// String-type JSONObjects convert only to string.
func (suite *JSONObjectSuite) TestConversionsString(c *C) {
	obj := jsonString("Test string")

	value, err := obj.GetString()
	c.Check(err, IsNil)
	c.Check(value, Equals, "Test string")

	_, err = obj.GetFloat64()
	c.Check(err, NotNil)
	_, err = obj.GetMap()
	c.Check(err, NotNil)
	_, err = obj.GetMAASObject()
	c.Check(err, NotNil)
	_, err = obj.GetArray()
	c.Check(err, NotNil)
	_, err = obj.GetBool()
	c.Check(err, NotNil)
}

// Number-type JSONObjects convert only to float64.
func (suite *JSONObjectSuite) TestConversionsFloat64(c *C) {
	obj := jsonFloat64(1.1)

	value, err := obj.GetFloat64()
	c.Check(err, IsNil)
	c.Check(value, Equals, 1.1)

	_, err = obj.GetString()
	c.Check(err, NotNil)
	_, err = obj.GetMap()
	c.Check(err, NotNil)
	_, err = obj.GetMAASObject()
	c.Check(err, NotNil)
	_, err = obj.GetArray()
	c.Check(err, NotNil)
	_, err = obj.GetBool()
	c.Check(err, NotNil)
}

// Map-type JSONObjects convert only to map.
func (suite *JSONObjectSuite) TestConversionsMap(c *C) {
	input := map[string]JSONObject{"x": jsonString("y")}
	obj := jsonMap(input)

	value, err := obj.GetMap()
	c.Check(err, IsNil)
	text, err := value["x"].GetString()
	c.Check(err, IsNil)
	c.Check(text, Equals, "y")

	_, err = obj.GetString()
	c.Check(err, NotNil)
	_, err = obj.GetFloat64()
	c.Check(err, NotNil)
	_, err = obj.GetMAASObject()
	c.Check(err, NotNil)
	_, err = obj.GetArray()
	c.Check(err, NotNil)
	_, err = obj.GetBool()
	c.Check(err, NotNil)
}

// Array-type JSONObjects convert only to array.
func (suite *JSONObjectSuite) TestConversionsArray(c *C) {
	obj := jsonArray([]JSONObject{jsonString("item")})

	value, err := obj.GetArray()
	c.Check(err, IsNil)
	text, err := value[0].GetString()
	c.Check(err, IsNil)
	c.Check(text, Equals, "item")

	_, err = obj.GetString()
	c.Check(err, NotNil)
	_, err = obj.GetFloat64()
	c.Check(err, NotNil)
	_, err = obj.GetMap()
	c.Check(err, NotNil)
	_, err = obj.GetMAASObject()
	c.Check(err, NotNil)
	_, err = obj.GetBool()
	c.Check(err, NotNil)
}

// Boolean-type JSONObjects convert only to bool.
func (suite *JSONObjectSuite) TestConversionsBool(c *C) {
	obj := jsonBool(false)

	value, err := obj.GetBool()
	c.Check(err, IsNil)
	c.Check(value, Equals, false)

	_, err = obj.GetString()
	c.Check(err, NotNil)
	_, err = obj.GetFloat64()
	c.Check(err, NotNil)
	_, err = obj.GetMap()
	c.Check(err, NotNil)
	_, err = obj.GetMAASObject()
	c.Check(err, NotNil)
	_, err = obj.GetArray()
	c.Check(err, NotNil)
}
