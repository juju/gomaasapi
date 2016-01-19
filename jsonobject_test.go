// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"

	. "gopkg.in/check.v1"
)

type JSONObjectSuite struct {
}

var _ = Suite(&JSONObjectSuite{})

// maasify() converts nil.
func (suite *JSONObjectSuite) TestMaasifyConvertsNil(c *C) {
	c.Check(maasify(Client{}, nil).IsNil(), Equals, true)
}

// maasify() converts strings.
func (suite *JSONObjectSuite) TestMaasifyConvertsString(c *C) {
	const text = "Hello"
	out, err := maasify(Client{}, text).GetString()
	c.Assert(err, IsNil)
	c.Check(out, Equals, text)
}

// maasify() converts float64 numbers.
func (suite *JSONObjectSuite) TestMaasifyConvertsNumber(c *C) {
	const number = 3.1415926535
	num, err := maasify(Client{}, number).GetFloat64()
	c.Assert(err, IsNil)
	c.Check(num, Equals, number)
}

// maasify() converts array slices.
func (suite *JSONObjectSuite) TestMaasifyConvertsArray(c *C) {
	original := []interface{}{3.0, 2.0, 1.0}
	output, err := maasify(Client{}, original).GetArray()
	c.Assert(err, IsNil)
	c.Check(len(output), Equals, len(original))
}

// When maasify() converts an array slice, the result contains JSONObjects.
func (suite *JSONObjectSuite) TestMaasifyArrayContainsJSONObjects(c *C) {
	arr, err := maasify(Client{}, []interface{}{9.9}).GetArray()
	c.Assert(err, IsNil)
	var _ JSONObject = arr[0]
	entry, err := arr[0].GetFloat64()
	c.Assert(err, IsNil)
	c.Check(entry, Equals, 9.9)
}

// maasify() converts maps.
func (suite *JSONObjectSuite) TestMaasifyConvertsMap(c *C) {
	original := map[string]interface{}{"1": "one", "2": "two", "3": "three"}
	output, err := maasify(Client{}, original).GetMap()
	c.Assert(err, IsNil)
	c.Check(len(output), Equals, len(original))
}

// When maasify() converts a map, the result contains JSONObjects.
func (suite *JSONObjectSuite) TestMaasifyMapContainsJSONObjects(c *C) {
	jsonobj := maasify(Client{}, map[string]interface{}{"key": "value"})
	mp, err := jsonobj.GetMap()
	var _ JSONObject = mp["key"]
	c.Assert(err, IsNil)
	entry, err := mp["key"].GetString()
	c.Check(entry, Equals, "value")
}

// maasify() converts MAAS objects.
func (suite *JSONObjectSuite) TestMaasifyConvertsMAASObject(c *C) {
	original := map[string]interface{}{
		"resource_uri": "http://example.com/foo",
		"size":         "3",
	}
	obj, err := maasify(Client{}, original).GetMAASObject()
	c.Assert(err, IsNil)
	c.Check(len(obj.GetMap()), Equals, len(original))
	size, err := obj.GetMap()["size"].GetString()
	c.Assert(err, IsNil)
	c.Check(size, Equals, "3")
}

// maasify() passes its client to a MAASObject it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesClientToMAASObject(c *C) {
	client := Client{}
	original := map[string]interface{}{"resource_uri": "/foo"}
	output, err := maasify(client, original).GetMAASObject()
	c.Assert(err, IsNil)
	c.Check(output.client, Equals, client)
}

// maasify() passes its client into an array of MAASObjects it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesClientIntoArray(c *C) {
	client := Client{}
	obj := map[string]interface{}{"resource_uri": "/foo"}
	list := []interface{}{obj}
	jsonobj, err := maasify(client, list).GetArray()
	c.Assert(err, IsNil)
	out, err := jsonobj[0].GetMAASObject()
	c.Assert(err, IsNil)
	c.Check(out.client, Equals, client)
}

// maasify() passes its client into a map of MAASObjects it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesClientIntoMap(c *C) {
	client := Client{}
	obj := map[string]interface{}{"resource_uri": "/foo"}
	mp := map[string]interface{}{"key": obj}
	jsonobj, err := maasify(client, mp).GetMap()
	c.Assert(err, IsNil)
	out, err := jsonobj["key"].GetMAASObject()
	c.Assert(err, IsNil)
	c.Check(out.client, Equals, client)
}

// maasify() passes its client all the way down into any MAASObjects in the
// object structure it creates.
func (suite *JSONObjectSuite) TestMaasifyPassesClientAllTheWay(c *C) {
	client := Client{}
	obj := map[string]interface{}{"resource_uri": "/foo"}
	mp := map[string]interface{}{"key": obj}
	list := []interface{}{mp}
	jsonobj, err := maasify(client, list).GetArray()
	c.Assert(err, IsNil)
	outerMap, err := jsonobj[0].GetMap()
	c.Assert(err, IsNil)
	out, err := outerMap["key"].GetMAASObject()
	c.Assert(err, IsNil)
	c.Check(out.client, Equals, client)
}

// maasify() converts Booleans.
func (suite *JSONObjectSuite) TestMaasifyConvertsBool(c *C) {
	t, err := maasify(Client{}, true).GetBool()
	c.Assert(err, IsNil)
	f, err := maasify(Client{}, false).GetBool()
	c.Assert(err, IsNil)
	c.Check(t, Equals, true)
	c.Check(f, Equals, false)
}

// Parse takes you from a JSON blob to a JSONObject.
func (suite *JSONObjectSuite) TestParseMaasifiesJSONBlob(c *C) {
	blob := []byte("[12]")
	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)

	arr, err := obj.GetArray()
	c.Assert(err, IsNil)
	out, err := arr[0].GetFloat64()
	c.Assert(err, IsNil)
	c.Check(out, Equals, 12.0)
}

func (suite *JSONObjectSuite) TestParseKeepsBinaryOriginal(c *C) {
	blob := []byte(`"Hi"`)

	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)

	text, err := obj.GetString()
	c.Assert(err, IsNil)
	c.Check(text, Equals, "Hi")
	binary, err := obj.GetBytes()
	c.Assert(err, IsNil)
	c.Check(binary, DeepEquals, blob)
}

func (suite *JSONObjectSuite) TestParseTreatsInvalidJSONAsBinary(c *C) {
	blob := []byte("?x]}y![{z")

	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)

	c.Check(obj.IsNil(), Equals, false)
	c.Check(obj.value, IsNil)
	binary, err := obj.GetBytes()
	c.Assert(err, IsNil)
	c.Check(binary, DeepEquals, blob)
}

func (suite *JSONObjectSuite) TestParseTreatsInvalidUTF8AsBinary(c *C) {
	// Arbitrary data that is definitely not UTF-8.
	blob := []byte{220, 8, 129}

	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)

	c.Check(obj.IsNil(), Equals, false)
	c.Check(obj.value, IsNil)
	binary, err := obj.GetBytes()
	c.Assert(err, IsNil)
	c.Check(binary, DeepEquals, blob)
}

func (suite *JSONObjectSuite) TestParseTreatsEmptyJSONAsBinary(c *C) {
	blob := []byte{}

	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)

	c.Check(obj.IsNil(), Equals, false)
	data, err := obj.GetBytes()
	c.Assert(err, IsNil)
	c.Check(data, DeepEquals, blob)
}

func (suite *JSONObjectSuite) TestParsePanicsOnNilJSON(c *C) {
	defer func() {
		failure := recover()
		c.Assert(failure, NotNil)
		c.Check(failure.(error).Error(), Matches, ".*nil input")
	}()
	Parse(Client{}, nil)
}

func (suite *JSONObjectSuite) TestParseNullProducesIsNil(c *C) {
	blob := []byte("null")
	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)
	c.Check(obj.IsNil(), Equals, true)
}

func (suite *JSONObjectSuite) TestParseNonNullProducesNonIsNil(c *C) {
	blob := []byte("1")
	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)
	c.Check(obj.IsNil(), Equals, false)
}

func (suite *JSONObjectSuite) TestParseSpacedNullProducesIsNil(c *C) {
	blob := []byte("      null     ")
	obj, err := Parse(Client{}, blob)
	c.Assert(err, IsNil)
	c.Check(obj.IsNil(), Equals, true)
}

// String-type JSONObjects convert only to string.
func (suite *JSONObjectSuite) TestConversionsString(c *C) {
	obj := maasify(Client{}, "Test string")

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
	obj := maasify(Client{}, 1.1)

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
	obj := maasify(Client{}, map[string]interface{}{"x": "y"})

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
	obj := maasify(Client{}, []interface{}{"item"})

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
	obj := maasify(Client{}, false)

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

func (suite *JSONObjectSuite) TestNilSerializesToJSON(c *C) {
	output, err := json.Marshal(maasify(Client{}, nil))
	c.Assert(err, IsNil)
	c.Check(output, DeepEquals, []byte("null"))
}

func (suite *JSONObjectSuite) TestEmptyStringSerializesToJSON(c *C) {
	output, err := json.Marshal(maasify(Client{}, ""))
	c.Assert(err, IsNil)
	c.Check(string(output), Equals, `""`)
}

func (suite *JSONObjectSuite) TestStringSerializesToJSON(c *C) {
	text := "Text wrapped in JSON"
	output, err := json.Marshal(maasify(Client{}, text))
	c.Assert(err, IsNil)
	c.Check(output, DeepEquals, []byte(fmt.Sprintf(`"%s"`, text)))
}

func (suite *JSONObjectSuite) TestStringIsEscapedInJSON(c *C) {
	text := `\"Quote,\" \\backslash, and \'apostrophe\'.`
	output, err := json.Marshal(maasify(Client{}, text))
	c.Assert(err, IsNil)
	var deserialized string
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized, Equals, text)
}

func (suite *JSONObjectSuite) TestFloat64SerializesToJSON(c *C) {
	number := 3.1415926535
	output, err := json.Marshal(maasify(Client{}, number))
	c.Assert(err, IsNil)
	var deserialized float64
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized, Equals, number)
}

func (suite *JSONObjectSuite) TestEmptyMapSerializesToJSON(c *C) {
	mp := map[string]interface{}{}
	output, err := json.Marshal(maasify(Client{}, mp))
	c.Assert(err, IsNil)
	var deserialized interface{}
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized.(map[string]interface{}), DeepEquals, mp)
}

func (suite *JSONObjectSuite) TestMapSerializesToJSON(c *C) {
	// Sample data: counting in Japanese.
	mp := map[string]interface{}{"one": "ichi", "two": "nii", "three": "san"}
	output, err := json.Marshal(maasify(Client{}, mp))
	c.Assert(err, IsNil)
	var deserialized interface{}
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized.(map[string]interface{}), DeepEquals, mp)
}

func (suite *JSONObjectSuite) TestEmptyArraySerializesToJSON(c *C) {
	arr := []interface{}{}
	output, err := json.Marshal(maasify(Client{}, arr))
	c.Assert(err, IsNil)
	var deserialized interface{}
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	// The deserialized value is a slice, and it contains no elements.
	// Can't do a regular comparison here because at least in the current
	// json implementation, an empty list deserializes as a nil slice,
	// not as an empty slice!
	// (It doesn't work that way for maps though, for some reason).
	c.Check(len(deserialized.([]interface{})), Equals, len(arr))
}

func (suite *JSONObjectSuite) TestArrayOfStringsSerializesToJSON(c *C) {
	value := "item"
	output, err := json.Marshal(maasify(Client{}, []interface{}{value}))
	c.Assert(err, IsNil)
	var deserialized []string
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized, DeepEquals, []string{value})
}

func (suite *JSONObjectSuite) TestArrayOfNumbersSerializesToJSON(c *C) {
	value := 9.0
	output, err := json.Marshal(maasify(Client{}, []interface{}{value}))
	c.Assert(err, IsNil)
	var deserialized []float64
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized, DeepEquals, []float64{value})
}

func (suite *JSONObjectSuite) TestArrayPreservesOrderInJSON(c *C) {
	// Sample data: counting in Korean.
	arr := []interface{}{"jong", "il", "ee", "sam"}
	output, err := json.Marshal(maasify(Client{}, arr))
	c.Assert(err, IsNil)

	var deserialized []interface{}
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized, DeepEquals, arr)
}

func (suite *JSONObjectSuite) TestBoolSerializesToJSON(c *C) {
	f, err := json.Marshal(maasify(Client{}, false))
	c.Assert(err, IsNil)
	t, err := json.Marshal(maasify(Client{}, true))
	c.Assert(err, IsNil)

	c.Check(f, DeepEquals, []byte("false"))
	c.Check(t, DeepEquals, []byte("true"))
}
