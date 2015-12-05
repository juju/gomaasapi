// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	. "gopkg.in/check.v1"
)

type MyTestSuite struct{}

var _ = Suite(&MyTestSuite{})

// TODO: Replace with real test functions.  Give them real names.
func (suite *MyTestSuite) TestXXX(c *C) {
	c.Check(2+2, Equals, 4)
}
