// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"

	. "gopkg.in/check.v1"
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

	input := map[string]interface{}{resourceURI: "%z"}
	newJSONMAASObject(input, Client{})
}

func (suite *MAASObjectSuite) TestNewJSONMAASObjectSetsUpURI(c *C) {
	URI, err := url.Parse("http://example.com/a/resource")
	c.Assert(err, IsNil)
	attrs := map[string]interface{}{resourceURI: URI.String()}
	obj := newJSONMAASObject(attrs, Client{})
	c.Check(obj.uri, DeepEquals, URI)
}

func (suite *MAASObjectSuite) TestURL(c *C) {
	baseURL, err := url.Parse("http://example.com/")
	c.Assert(err, IsNil)
	uri := "http://example.com/a/resource"
	resourceURL, err := url.Parse(uri)
	c.Assert(err, IsNil)
	input := map[string]interface{}{resourceURI: uri}
	client := Client{APIURL: baseURL}
	obj := newJSONMAASObject(input, client)

	URL := obj.URL()

	c.Check(URL, DeepEquals, resourceURL)
}

// makeFakeMAASObject creates a MAASObject for some imaginary resource.
// There is no actual HTTP service or resource attached.
// serviceURL is the base URL of the service, and resourceURI is the path for
// the object, relative to serviceURL.
func makeFakeMAASObject(serviceURL, resourcePath string) MAASObject {
	baseURL, err := url.Parse(serviceURL)
	if err != nil {
		panic(fmt.Errorf("creation of fake object failed: %v", err))
	}
	uri := serviceURL + resourcePath
	input := map[string]interface{}{resourceURI: uri}
	client := Client{APIURL: baseURL}
	return newJSONMAASObject(input, client)
}

// Passing GetSubObject a relative path effectively concatenates that path to
// the original object's resource URI.
func (suite *MAASObjectSuite) TestGetSubObjectRelative(c *C) {
	obj := makeFakeMAASObject("http://example.com/", "a/resource/")

	subObj := obj.GetSubObject("test")
	subURL := subObj.URL()

	// uri ends with a slash and subName starts with one, but the two paths
	// should be concatenated as "http://example.com/a/resource/test/".
	expectedSubURL, err := url.Parse("http://example.com/a/resource/test/")
	c.Assert(err, IsNil)
	c.Check(subURL, DeepEquals, expectedSubURL)
}

// Passing GetSubObject an absolute path effectively substitutes that path for
// the path component in the original object's resource URI.
func (suite *MAASObjectSuite) TestGetSubObjectAbsolute(c *C) {
	obj := makeFakeMAASObject("http://example.com/", "a/resource/")

	subObj := obj.GetSubObject("/b/test")
	subURL := subObj.URL()

	expectedSubURL, err := url.Parse("http://example.com/b/test/")
	c.Assert(err, IsNil)
	c.Check(subURL, DeepEquals, expectedSubURL)
}

// An absolute path passed to GetSubObject is rooted at the server root, not
// at the service root.  So every absolute resource URI must repeat the part
// of the path that leads to the service root.  This does not double that part
// of the URI.
func (suite *MAASObjectSuite) TestGetSubObjectAbsoluteDoesNotDoubleServiceRoot(c *C) {
	obj := makeFakeMAASObject("http://example.com/service", "a/resource/")

	subObj := obj.GetSubObject("/service/test")
	subURL := subObj.URL()

	// The "/service" part is not repeated; it must be included.
	expectedSubURL, err := url.Parse("http://example.com/service/test/")
	c.Assert(err, IsNil)
	c.Check(subURL, DeepEquals, expectedSubURL)
}

// The argument to GetSubObject is a relative path, not a URL.  So it won't
// take a query part.  The special characters that mark a query are escaped
// so they are recognized as parts of the path.
func (suite *MAASObjectSuite) TestGetSubObjectTakesPathNotURL(c *C) {
	obj := makeFakeMAASObject("http://example.com/", "x/")

	subObj := obj.GetSubObject("/y?z")

	c.Check(subObj.URL().String(), Equals, "http://example.com/y%3Fz/")
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
