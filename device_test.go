// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"net/http"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type deviceSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&deviceSuite{})

func (*deviceSuite) TestReadDevicesBadSchema(c *gc.C) {
	_, err := readDevices(twoDotOh, "wat?")
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `device base schema check failed: expected list, got string("wat?")`)
}

func (*deviceSuite) TestReadDevices(c *gc.C) {
	devices, err := readDevices(twoDotOh, parseJSON(c, devicesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(devices, gc.HasLen, 1)

	device := devices[0]
	c.Assert(device.SystemID(), gc.Equals, "4y3ha8")
	c.Assert(device.Hostname(), gc.Equals, "furnacelike-brittney")
	c.Assert(device.FQDN(), gc.Equals, "furnacelike-brittney.maas")
	c.Assert(device.IPAddresses(), jc.DeepEquals, []string{"192.168.100.11"})
	zone := device.Zone()
	c.Assert(zone, gc.NotNil)
	c.Assert(zone.Name(), gc.Equals, "default")
}

func (*deviceSuite) TestLowVersion(c *gc.C) {
	_, err := readDevices(version.MustParse("1.9.0"), parseJSON(c, devicesResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (*deviceSuite) TestHighVersion(c *gc.C) {
	devices, err := readDevices(version.MustParse("2.1.9"), parseJSON(c, devicesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(devices, gc.HasLen, 1)
}

func (s *deviceSuite) setupDelete(c *gc.C) (*SimpleTestServer, *device) {
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)

	devices, err := controller.Devices(DevicesArgs{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(devices, gc.HasLen, 1)
	return server, devices[0].(*device)
}

func (s *deviceSuite) TestDelete(c *gc.C) {
	server, device := s.setupDelete(c)
	// Successful delete is 204 - StatusNoContent
	server.AddDeleteResponse(device.resourceURI, http.StatusNoContent, "")
	err := device.Delete()
	c.Assert(err, jc.ErrorIsNil)
}

func (s *deviceSuite) TestDelete404(c *gc.C) {
	_, device := s.setupDelete(c)
	// No path, so 404
	err := device.Delete()
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (s *deviceSuite) TestDeleteForbidden(c *gc.C) {
	server, device := s.setupDelete(c)
	server.AddDeleteResponse(device.resourceURI, http.StatusForbidden, "")
	err := device.Delete()
	c.Assert(err, jc.Satisfies, IsPermissionError)
}

func (s *deviceSuite) TestDeleteUnknown(c *gc.C) {
	server, device := s.setupDelete(c)
	server.AddDeleteResponse(device.resourceURI, http.StatusConflict, "")
	err := device.Delete()
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
}

const (
	deviceResponse = `
        {
            "domain": {
                "name": "maas",
                "ttl": null,
                "resource_record_count": 0,
                "resource_uri": "/MAAS/api/2.0/domains/0/",
                "id": 0,
                "authoritative": true
            },
            "tag_names": [],
            "hostname": "furnacelike-brittney",
            "zone": {
                "name": "default",
                "description": "",
                "resource_uri": "/MAAS/api/2.0/zones/default/"
            },
            "parent": null,
            "system_id": "4y3ha8",
            "node_type": 1,
            "ip_addresses": ["192.168.100.11"],
            "resource_uri": "/MAAS/api/2.0/devices/4y3ha8/",
            "owner": "thumper",
            "fqdn": "furnacelike-brittney.maas",
            "node_type_name": "Device",
            "macaddress_set": [
                {
                    "mac_address": "b8:6a:6d:58:b3:7d"
                }
            ]
        }
        `
	devicesResponse = "[" + deviceResponse + "]"
)
