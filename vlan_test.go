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

func (*vlanSuite) TestReadVLANsBadSchema(c *gc.C) {
	_, err := readVLANs(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `vlan base schema check failed: expected list, got string("wat?")`)
}

func (s *vlanSuite) TestReadVLANsWithName(c *gc.C) {
	vlans, err := readVLANs(twoDotOh, parseJSON(c, vlanResponseWithName))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
	readVLAN := vlans[0]
	s.assertVLAN(c, readVLAN, &vlan{
		id:            1,
		name:          "untagged",
		fabric:        "fabric-0",
		vid:           2,
		mtu:           1500,
		dhcp:          true,
		primaryRack:   "a-rack",
		secondaryRack: "",
	})
}

func (*vlanSuite) assertVLAN(c *gc.C, givenVLAN, expectedVLAN *vlan) {
	c.Check(givenVLAN.ID(), gc.Equals, expectedVLAN.id)
	c.Check(givenVLAN.Name(), gc.Equals, expectedVLAN.name)
	c.Check(givenVLAN.Fabric(), gc.Equals, expectedVLAN.fabric)
	c.Check(givenVLAN.VID(), gc.Equals, expectedVLAN.vid)
	c.Check(givenVLAN.MTU(), gc.Equals, expectedVLAN.mtu)
	c.Check(givenVLAN.DHCP(), gc.Equals, expectedVLAN.dhcp)
	c.Check(givenVLAN.PrimaryRack(), gc.Equals, expectedVLAN.primaryRack)
	c.Check(givenVLAN.SecondaryRack(), gc.Equals, expectedVLAN.secondaryRack)
}

func (s *vlanSuite) TestReadVLANsWithoutName(c *gc.C) {
	vlans, err := readVLANs(twoDotOh, parseJSON(c, vlanResponseWithoutName))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
	readVLAN := vlans[0]
	s.assertVLAN(c, readVLAN, &vlan{
		id:            5006,
		name:          "",
		fabric:        "maas-management",
		vid:           30,
		mtu:           1500,
		dhcp:          true,
		primaryRack:   "4y3h7n",
		secondaryRack: "",
	})
}

func (*vlanSuite) TestLowVersion(c *gc.C) {
	_, err := readVLANs(version.MustParse("1.9.0"), parseJSON(c, vlanResponseWithName))
	c.Assert(err.Error(), gc.Equals, `no vlan read func for version 1.9.0`)
}

func (*vlanSuite) TestHighVersion(c *gc.C) {
	vlans, err := readVLANs(version.MustParse("2.1.9"), parseJSON(c, vlanResponseWithoutName))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
}

const (
	vlanResponseWithName = `
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
	vlanResponseWithoutName = `
[
    {
        "dhcp_on": true,
        "id": 5006,
        "mtu": 1500,
        "fabric": "maas-management",
        "vid": 30,
        "primary_rack": "4y3h7n",
        "name": null,
        "external_dhcp": null,
        "resource_uri": "/MAAS/api/2.0/vlans/5006/",
        "secondary_rack": null
    }
]
`
)
