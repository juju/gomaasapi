// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	. "launchpad.net/gocheck"
	"net/url"
)

func (suite *GomaasapiTestSuite) TestNewMAASUsesBaseURLFromClient(c *C) {
	baseURLString := "https://server.com:888/path/to/api"
	baseURL, _ := url.Parse(baseURLString)
	client := Client{BaseURL: baseURL}
	maas, err := NewMAAS(client)
	c.Check(err, IsNil)
	URL, err := maas.URL()
	c.Check(err, IsNil)
	c.Check(URL, DeepEquals, baseURL)
}
