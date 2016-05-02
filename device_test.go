// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"net/http"

	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type deviceSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&deviceSuite{})

func (*deviceSuite) TestNilZone(c *gc.C) {
	var empty device
	c.Check(empty.Zone() == nil, jc.IsTrue)
}

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
	c.Check(device.SystemID(), gc.Equals, "4y3haf")
	c.Check(device.Hostname(), gc.Equals, "furnacelike-brittney")
	c.Check(device.FQDN(), gc.Equals, "furnacelike-brittney.maas")
	c.Check(device.IPAddresses(), jc.DeepEquals, []string{"192.168.100.11"})
	zone := device.Zone()
	c.Check(zone, gc.NotNil)
	c.Check(zone.Name(), gc.Equals, "default")
}

func (*deviceSuite) TestReadDevicesNils(c *gc.C) {
	json := parseJSON(c, devicesResponse)
	deviceMap := json.([]interface{})[0].(map[string]interface{})
	deviceMap["owner"] = nil
	deviceMap["parent"] = nil
	devices, err := readDevices(twoDotOh, json)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(devices, gc.HasLen, 1)

	device := devices[0]
	c.Check(device.Owner(), gc.Equals, "")
	c.Check(device.Parent(), gc.Equals, "")
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

func (s *deviceSuite) TestInterfaceSet(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddGetResponse(device.interfacesURI(), http.StatusOK, interfacesResponse)
	ifaces := device.InterfaceSet()
	c.Assert(ifaces, gc.HasLen, 2)
}

type fakeVLAN struct {
	VLAN
	id int
}

func (f *fakeVLAN) ID() int {
	return f.id
}

func (s *controllerSuite) TestCreateInterfaceArgsValidate(c *gc.C) {
	for i, test := range []struct {
		args    CreateInterfaceArgs
		errText string
	}{{
		errText: "missing Name not valid",
	}, {
		args:    CreateInterfaceArgs{Name: "eth3"},
		errText: "missing MACAddress not valid",
	}, {
		args:    CreateInterfaceArgs{Name: "eth3", MACAddress: "a-mac-address"},
		errText: `missing VLAN not valid`,
	}, {
		args: CreateInterfaceArgs{Name: "eth3", MACAddress: "a-mac-address", VLAN: &fakeVLAN{}},
	}} {
		c.Logf("test %d", i)
		err := test.args.Validate()
		if test.errText == "" {
			c.Check(err, jc.ErrorIsNil)
		} else {
			c.Check(err, jc.Satisfies, errors.IsNotValid)
			c.Check(err.Error(), gc.Equals, test.errText)
		}
	}
}

func (s *deviceSuite) TestCreateInterfaceValidates(c *gc.C) {
	_, device := s.getServerAndDevice(c)
	_, err := device.CreateInterface(CreateInterfaceArgs{})
	c.Assert(err, jc.Satisfies, errors.IsNotValid)
}

func (s *deviceSuite) TestCreateInterface(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusOK, interfaceResponse)

	iface, err := device.CreateInterface(CreateInterfaceArgs{
		Name:       "eth43",
		MACAddress: "some-mac-address",
		VLAN:       &fakeVLAN{id: 33},
		Tags:       []string{"foo", "bar"},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(iface, gc.NotNil)

	request := server.LastRequest()
	form := request.PostForm
	c.Assert(form.Get("name"), gc.Equals, "eth43")
	c.Assert(form.Get("mac_address"), gc.Equals, "some-mac-address")
	c.Assert(form.Get("vlan"), gc.Equals, "33")
	c.Assert(form.Get("tags"), gc.Equals, "foo,bar")
}

func minimalCreateInterfaceArgs() CreateInterfaceArgs {
	return CreateInterfaceArgs{
		Name:       "eth43",
		MACAddress: "some-mac-address",
		VLAN:       &fakeVLAN{id: 33},
	}
}

func (s *deviceSuite) TestCreateInterfaceNotFound(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusNotFound, "can't find device")
	_, err := device.CreateInterface(minimalCreateInterfaceArgs())
	c.Assert(err, jc.Satisfies, IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "can't find device")
}

func (s *deviceSuite) TestCreateInterfaceConflict(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusConflict, "device not allocated")
	_, err := device.CreateInterface(minimalCreateInterfaceArgs())
	c.Assert(err, jc.Satisfies, IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "device not allocated")
}

func (s *deviceSuite) TestCreateInterfaceForbidden(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusForbidden, "device not yours")
	_, err := device.CreateInterface(minimalCreateInterfaceArgs())
	c.Assert(err, jc.Satisfies, IsPermissionError)
	c.Assert(err.Error(), gc.Equals, "device not yours")
}

func (s *deviceSuite) TestCreateInterfaceServiceUnavailable(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusServiceUnavailable, "no ip addresses available")
	_, err := device.CreateInterface(minimalCreateInterfaceArgs())
	c.Assert(err, jc.Satisfies, IsCannotCompleteError)
	c.Assert(err.Error(), gc.Equals, "no ip addresses available")
}

func (s *deviceSuite) TestCreateInterfaceUnknown(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusMethodNotAllowed, "wat?")
	_, err := device.CreateInterface(minimalCreateInterfaceArgs())
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func (s *deviceSuite) getServerAndDevice(c *gc.C) (*SimpleTestServer, *device) {
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)

	devices, err := controller.Devices(DevicesArgs{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(devices, gc.HasLen, 1)
	return server, devices[0].(*device)
}

func (s *deviceSuite) TestDelete(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	// Successful delete is 204 - StatusNoContent
	server.AddDeleteResponse(device.resourceURI, http.StatusNoContent, "")
	err := device.Delete()
	c.Assert(err, jc.ErrorIsNil)
}

func (s *deviceSuite) TestDelete404(c *gc.C) {
	_, device := s.getServerAndDevice(c)
	// No path, so 404
	err := device.Delete()
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (s *deviceSuite) TestDeleteForbidden(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddDeleteResponse(device.resourceURI, http.StatusForbidden, "")
	err := device.Delete()
	c.Assert(err, jc.Satisfies, IsPermissionError)
}

func (s *deviceSuite) TestDeleteUnknown(c *gc.C) {
	server, device := s.getServerAndDevice(c)
	server.AddDeleteResponse(device.resourceURI, http.StatusConflict, "")
	err := device.Delete()
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
}

const (
	deviceResponse = `
    {
        "zone": {
            "description": "",
            "resource_uri": "/MAAS/api/2.0/zones/default/",
            "name": "default"
        },
        "domain": {
            "resource_record_count": 0,
            "resource_uri": "/MAAS/api/2.0/domains/0/",
            "authoritative": true,
            "name": "maas",
            "ttl": null,
            "id": 0
        },
        "node_type_name": "Device",
        "address_ttl": null,
        "hostname": "furnacelike-brittney",
        "node_type": 1,
        "resource_uri": "/MAAS/api/2.0/devices/4y3haf/",
        "ip_addresses": ["192.168.100.11"],
        "owner": "thumper",
        "tag_names": [],
        "fqdn": "furnacelike-brittney.maas",
        "system_id": "4y3haf",
        "parent": "4y3ha3",
        "interface_set": [
            {
                "resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/48/",
                "type": "physical",
                "mac_address": "78:f0:f1:16:a7:46",
                "params": "",
                "discovered": null,
                "effective_mtu": 1500,
                "id": 48,
                "children": [],
                "links": [],
                "name": "eth0",
                "vlan": {
                    "secondary_rack": null,
                    "dhcp_on": true,
                    "fabric": "fabric-0",
                    "mtu": 1500,
                    "primary_rack": "4y3h7n",
                    "resource_uri": "/MAAS/api/2.0/vlans/1/",
                    "external_dhcp": null,
                    "name": "untagged",
                    "id": 1,
                    "vid": 0
                },
                "tags": [],
                "parents": [],
                "enabled": true
            },
            {
                "resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/49/",
                "type": "physical",
                "mac_address": "15:34:d3:2d:f7:a7",
                "params": {},
                "discovered": null,
                "effective_mtu": 1500,
                "id": 49,
                "children": [],
                "links": [
                    {
                        "mode": "link_up",
                        "id": 101
                    }
                ],
                "name": "eth1",
                "vlan": {
                    "secondary_rack": null,
                    "dhcp_on": true,
                    "fabric": "fabric-0",
                    "mtu": 1500,
                    "primary_rack": "4y3h7n",
                    "resource_uri": "/MAAS/api/2.0/vlans/1/",
                    "external_dhcp": null,
                    "name": "untagged",
                    "id": 1,
                    "vid": 0
                },
                "tags": [],
                "parents": [],
                "enabled": true
            }
        ]
    }
    `
	devicesResponse = "[" + deviceResponse + "]"
)
