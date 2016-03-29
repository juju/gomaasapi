// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

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
