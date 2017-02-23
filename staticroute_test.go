// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type staticRouteSuite struct{}

var _ = gc.Suite(&staticRouteSuite{})

func (*staticRouteSuite) TestReadStaticRoutesBadSchema(c *gc.C) {
	_, err := readStaticRoutes(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `static-route base schema check failed: expected list, got string("wat?")`)
}

func (*staticRouteSuite) TestReadStaticRoutes(c *gc.C) {
	staticRoutes, err := readStaticRoutes(twoDotOh, parseJSON(c, staticRoutesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(staticRoutes, gc.HasLen, 1)

	staticRoute := staticRoutes[0]
	c.Assert(staticRoute.ID(), gc.Equals, 2)
	c.Assert(staticRoute.Metric(), gc.Equals, int(0))
	c.Assert(staticRoute.GatewayIP(), gc.Equals, "192.168.0.1")
	source := staticRoute.Source()
	c.Assert(source, gc.NotNil)
	c.Assert(source.Name(), gc.Equals, "192.168.0.0/24")
	c.Assert(source.CIDR(), gc.Equals, "192.168.0.0/24")
	destination := staticRoute.Destination()
	c.Assert(destination, gc.NotNil)
	c.Assert(destination.Name(), gc.Equals, "Local-192")
	c.Assert(destination.CIDR(), gc.Equals, "192.168.0.0/16")
}

func (*staticRouteSuite) TestLowVersion(c *gc.C) {
	_, err := readStaticRoutes(version.MustParse("1.9.0"), parseJSON(c, staticRoutesResponse))
	c.Assert(err.Error(), gc.Equals, `no static-route read func for version 1.9.0`)
}

func (*staticRouteSuite) TestHighVersion(c *gc.C) {
	staticRoutes, err := readStaticRoutes(version.MustParse("2.1.9"), parseJSON(c, staticRoutesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(staticRoutes, gc.HasLen, 1)
}

var staticRoutesResponse = `
[
    {
        "destination": {
            "active_discovery": false,
            "id": 3,
            "resource_uri": "/MAAS/api/2.0/subnets/3/",
            "allow_proxy": true,
            "rdns_mode": 2,
            "dns_servers": [
                "8.8.8.8"
            ],
            "name": "Local-192",
            "cidr": "192.168.0.0/16",
            "space": "space-0",
            "vlan": {
                "fabric": "fabric-1",
                "id": 5002,
                "dhcp_on": false,
                "primary_rack": null,
                "resource_uri": "/MAAS/api/2.0/vlans/5002/",
                "mtu": 1500,
                "fabric_id": 1,
                "secondary_rack": null,
                "name": "untagged",
                "external_dhcp": null,
                "vid": 0
            },
            "gateway_ip": "192.168.0.1"
        },
        "source": {
            "active_discovery": false,
            "id": 1,
            "resource_uri": "/MAAS/api/2.0/subnets/1/",
            "allow_proxy": true,
            "rdns_mode": 2,
            "dns_servers": [],
            "name": "192.168.0.0/24",
            "cidr": "192.168.0.0/24",
            "space": "space-0",
            "vlan": {
                "fabric": "fabric-0",
                "id": 5001,
                "dhcp_on": false,
                "primary_rack": null,
                "resource_uri": "/MAAS/api/2.0/vlans/5001/",
                "mtu": 1500,
                "fabric_id": 0,
                "secondary_rack": null,
                "name": "untagged",
                "external_dhcp": "192.168.0.1",
                "vid": 0
            },
            "gateway_ip": null
        },
        "id": 2,
        "resource_uri": "/MAAS/api/2.0/static-routes/2/",
        "metric": 0,
        "gateway_ip": "192.168.0.1"
    }
]
`
