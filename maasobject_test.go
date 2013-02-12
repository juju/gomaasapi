// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"
	. "launchpad.net/gocheck"
	"math/rand"
	"net/url"
)

type MAASObjectSuite struct{}

var _ = Suite(&MAASObjectSuite{})

func makeFakeResourceURI() string {
	return "http://example.com/" + fmt.Sprint(rand.Int31())
}

// JSONObjects containing MAAS objects convert only to map or to MAASObject.
func (suite *MAASObjectSuite) TestConversionsMAASObject(c *C) {
	input := map[string]interface{}{resourceURI: "someplace"}
	obj := maasify(Client{}, input)

	mp, err := obj.GetMap()
	c.Check(err, IsNil)
	text, err := mp[resourceURI].GetString()
	c.Check(err, IsNil)
	c.Check(text, Equals, "someplace")

	var maasobj MAASObject
	maasobj, err = obj.GetMAASObject()
	c.Assert(err, IsNil)
	c.Check(maasobj, NotNil)

	_, err = obj.GetString()
	c.Check(err, NotNil)
	_, err = obj.GetFloat64()
	c.Check(err, NotNil)
	_, err = obj.GetArray()
	c.Check(err, NotNil)
	_, err = obj.GetBool()
	c.Check(err, NotNil)
}

func (suite *MAASObjectSuite) TestNewJSONMAASObjectPanicsIfNoResourceURI(c *C) {
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		msg := recoveredError.(error).Error()
		c.Check(msg, Matches, ".*no 'resource_uri' key.*")
	}()

	input := map[string]interface{}{"test": "test"}
	newJSONMAASObject(input, Client{})
}

func (suite *MAASObjectSuite) TestNewJSONMAASObjectPanicsIfResourceURINotString(c *C) {
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		msg := recoveredError.(error).Error()
		c.Check(msg, Matches, ".*invalid resource_uri.*")
	}()

	input := map[string]interface{}{resourceURI: 77.77}
	newJSONMAASObject(input, Client{})
}

func (suite *MAASObjectSuite) TestNewJSONMAASObjectPanicsIfResourceURINotURL(c *C) {
	defer func() {
		recoveredError := recover()
		c.Check(recoveredError, NotNil)
		msg := recoveredError.(error).Error()
		c.Check(msg, Matches, ".*resource_uri.*valid URL.*")
	}()

	input := map[string]interface{}{resourceURI: ""}
	newJSONMAASObject(input, Client{})
}

func (suite *MAASObjectSuite) TestNewJSONMAASObjectSetsUpURI(c *C) {
	URI, _ := url.Parse("http://example.com/a/resource")
	attrs := map[string]interface{}{resourceURI: URI.String()}
	obj := newJSONMAASObject(attrs, Client{})
	c.Check(obj.uri, DeepEquals, URI)
}

func (suite *MAASObjectSuite) TestURL(c *C) {
	baseURL, _ := url.Parse("http://example.com/")
	uri := "http://example.com/a/resource"
	resourceURL, _ := url.Parse(uri)
	input := map[string]interface{}{resourceURI: uri}
	client := Client{BaseURL: baseURL}
	obj := newJSONMAASObject(input, client)

	URL := obj.URL()

	c.Check(URL, DeepEquals, resourceURL)
}

func (suite *MAASObjectSuite) TestGetSubObjectRelative(c *C) {
	baseURL, _ := url.Parse("http://example.com/")
	uri := "http://example.com/a/resource/"
	input := map[string]interface{}{resourceURI: uri}
	client := Client{BaseURL: baseURL}
	obj := newJSONMAASObject(input, client)
	subName := "test"

	subObj := obj.GetSubObject(subName)
	subURL := subObj.URL()

	// uri ends with a slash and subName starts with one, but the two paths
	// should be concatenated as "http://example.com/a/resource/test/".
	expectedSubURL, _ := url.Parse("http://example.com/a/resource/test/")
	c.Check(subURL, DeepEquals, expectedSubURL)
}

func (suite *MAASObjectSuite) TestGetSubObjectAbsolute(c *C) {
	baseURL, _ := url.Parse("http://example.com/")
	uri := "http://example.com/a/resource/"
	input := map[string]interface{}{resourceURI: uri}
	client := Client{BaseURL: baseURL}
	obj := newJSONMAASObject(input, client)
	subName := "/b/test"

	subObj := obj.GetSubObject(subName)
	subURL := subObj.URL()

	expectedSubURL, _ := url.Parse("http://example.com/b/test/")
	c.Check(subURL, DeepEquals, expectedSubURL)
}

func (suite *MAASObjectSuite) TestGetField(c *C) {
	uri := "http://example.com/a/resource"
	fieldName := "field name"
	fieldValue := "a value"
	input := map[string]interface{}{
		resourceURI: uri, fieldName: fieldValue,
	}
	obj := newJSONMAASObject(input, Client{})
	value, err := obj.GetField(fieldName)
	c.Check(err, IsNil)
	c.Check(value, Equals, fieldValue)
}

func (suite *MAASObjectSuite) TestSerializesToJSON(c *C) {
	attrs := map[string]interface{}{
		resourceURI: "http://maas.example.com/",
		"counter":   5.0,
		"active":    true,
		"macs":      map[string]interface{}{"eth0": "AA:BB:CC:DD:EE:FF"},
	}
	obj := maasify(Client{}, attrs)
	output, err := json.Marshal(obj)
	c.Assert(err, IsNil)

	var deserialized map[string]interface{}
	err = json.Unmarshal(output, &deserialized)
	c.Assert(err, IsNil)
	c.Check(deserialized, DeepEquals, attrs)
}
