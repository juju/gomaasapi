// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

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
