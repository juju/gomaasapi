// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"launchpad.net/gocheck"
)


// maasify() converts nil.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsNil(c *gocheck.C) {
	c.Check(maasify(nil), gocheck.Equals, nil)
}


// maasify() converts strings.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsString(c *gocheck.C) {
	const text = "Hello"
	c.Check(string(maasify(text).(maasString)), gocheck.Equals, text)
}


// maasify() converts float64 numbers.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsNumber(c *gocheck.C) {
	const number = 3.1415926535
	c.Check(float64(maasify(number).(maasFloat64)), gocheck.Equals, number)
}


// maasify() converts array slices.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsArray(c *gocheck.C) {
	original := []interface{}{3.0, 2.0, 1.0}
	output := maasify(original).(maasArray)
	c.Check(len(output), gocheck.Equals, len(original))
}


// maasify() converts maps.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsMap(c *gocheck.C) {
	original := map[string]interface{}{"1": "one", "2": "two", "3": "three"}
	output := maasify(original).(maasMap)
	c.Check(len(output), gocheck.Equals, len(original))
}


// maasify() converts Booleans.
func (suite *GomaasapiTestSuite) TestMaasifyConvertsBool(c *gocheck.C) {
	c.Check(bool(maasify(true).(maasBool)), gocheck.Equals, true)
	c.Check(bool(maasify(false).(maasBool)), gocheck.Equals, false)
}
