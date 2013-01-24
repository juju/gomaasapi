// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"launchpad.net/gocheck"
)


type MaasObjectSuite struct{}

var _ = gocheck.Suite(&MaasObjectSuite{})


func (suite *MaasObjectSuite) TestMaasifyConvertsNil(c *gocheck.C) {
	c.Check(maasify(nil), gocheck.Equals, nil)
}
