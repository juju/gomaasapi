// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type vlanSuite struct{}

var _ = gc.Suite(&vlanSuite{})

func (*vlanSuite) TestreadVLANsBadSchema(c *gc.C) {
	_, err := readVLANs(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `vlan base schema check failed: expected list, got string("wat?")`)
}

func (*vlanSuite) TestreadVLANs(c *gc.C) {
	vlans, err := readVLANs(twoDotOh, parseJSON(c, vlanResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
	vlan := vlans[0]
	c.Assert(vlan.ID(), gc.Equals, 1)
	c.Assert(vlan.Name(), gc.Equals, "untagged")
	c.Assert(vlan.Fabric(), gc.Equals, "fabric-0")
	c.Assert(vlan.VID(), gc.Equals, 2)
	c.Assert(vlan.MTU(), gc.Equals, 1500)
	c.Assert(vlan.DHCP(), jc.IsTrue)
	c.Assert(vlan.PrimaryRack(), gc.Equals, "a-rack")
	c.Assert(vlan.SecondaryRack(), gc.Equals, "")
}

func (*vlanSuite) TestLowVersion(c *gc.C) {
	_, err := readVLANs(version.MustParse("1.9.0"), parseJSON(c, vlanResponse))
	c.Assert(err.Error(), gc.Equals, `no vlan read func for version 1.9.0`)
}

func (*vlanSuite) TestHighVersion(c *gc.C) {
	vlans, err := readVLANs(version.MustParse("2.1.9"), parseJSON(c, vlanResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
}

var vlanResponse = `
[
    {
        "name": "untagged",
        "vid": 2,
        "primary_rack": "a-rack",
        "resource_uri": "/MAAS/api/2.0/vlans/1/",
        "id": 1,
        "secondary_rack": null,
        "fabric": "fabric-0",
        "mtu": 1500,
        "dhcp_on": true
    }
]
`
