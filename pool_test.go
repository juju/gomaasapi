// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type poolSuite struct{}

var _ = gc.Suite(&poolSuite{})

func (*poolSuite) TestReadPoolsBadSchema(c *gc.C) {
	_, err := readPools(twoDotOh, "blahfoob")
	c.Assert(err.Error(), gc.Equals, `pool base schema check failed: expected list, got string("blahfoob")`)
}

func (*poolSuite) TestReadPools(c *gc.C) {
	pools, err := readPools(twoDotOh, parseJSON(c, poolResponse))
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(pools, gc.HasLen, 2)

	c.Assert(pools[0].Name(), gc.Equals, "default")
	c.Assert(pools[0].Description(), gc.Equals, "default description")

	c.Assert(pools[1].Name(), gc.Equals, "swimming_is_fun")
	c.Assert(pools[1].Description(), gc.Equals, "swimming is fun description")
}

// Pools were not introduced until 2.5.x
func (*poolSuite) TestLowVersion(c *gc.C) {
	_, err := readPools(version.MustParse("1.9.0"), parseJSON(c, poolResponse))
	c.Assert(err.Error(), gc.Equals, `no pool read func for version 1.9.0`)
}

// MaaS 2.6.x is GA now.
func (*poolSuite) TestHighVersion(c *gc.C) {
	pools, err := readPools(version.MustParse("2.1.9"), parseJSON(c, poolResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(pools, gc.HasLen, 2)
}

var poolResponse = `
[
    {
        "description": "default description",
        "resource_uri": "/MAAS/api/2.0/pools/default/",
        "name": "default"
    }, {
        "description": "swimming is fun description",
        "resource_uri": "/MAAS/api/2.0/pools/swimming_is_fun/",
        "name": "swimming_is_fun"
    }
]
`
