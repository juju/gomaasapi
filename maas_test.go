// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"net/url"

	. "gopkg.in/check.v1"
)

type MAASSuite struct{}

var _ = Suite(&MAASSuite{})

func (suite *MAASSuite) TestNewMAASUsesBaseURLFromClient(c *C) {
	baseURLString := "https://server.com:888/"
	baseURL, _ := url.Parse(baseURLString)
	client := Client{APIURL: baseURL}
	maas := NewMAAS(client)
	URL := maas.URL()
	c.Check(URL, DeepEquals, baseURL)
}
