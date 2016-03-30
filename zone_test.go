// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type zoneSuite struct{}

var _ = gc.Suite(&zoneSuite{})

func (*zoneSuite) TestReadZonesBadSchema(c *gc.C) {
	_, err := readZones(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `zone base schema check failed: expected list, got string("wat?")`)
}

func (*zoneSuite) TestReadZones(c *gc.C) {
	zones, err := readZones(twoDotOh, parseJSON(c, zoneResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zones, gc.HasLen, 2)
	c.Assert(zones[0].Name(), gc.Equals, "default")
	c.Assert(zones[0].Description(), gc.Equals, "default description")
	c.Assert(zones[1].Name(), gc.Equals, "special")
	c.Assert(zones[1].Description(), gc.Equals, "special description")
}

func (*zoneSuite) TestLowVersion(c *gc.C) {
	_, err := readZones(version.MustParse("1.9.0"), parseJSON(c, zoneResponse))
	c.Assert(err.Error(), gc.Equals, `no zone read func for version 1.9.0`)
}

func (*zoneSuite) TestHighVersion(c *gc.C) {
	zones, err := readZones(version.MustParse("2.1.9"), parseJSON(c, zoneResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zones, gc.HasLen, 2)
}

var zoneResponse = `
[
    {
        "description": "default description",
        "resource_uri": "/MAAS/api/2.0/zones/default/",
        "name": "default"
    }, {
        "description": "special description",
        "resource_uri": "/MAAS/api/2.0/zones/special/",
        "name": "special"
    }
]
`
