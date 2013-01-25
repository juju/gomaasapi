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
