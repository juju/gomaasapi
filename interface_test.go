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

type interfaceSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&interfaceSuite{})

func (*interfaceSuite) TestReadInterfacesBadSchema(c *gc.C) {
	_, err := readInterfaces(twoDotOh, "wat?")
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `interface base schema check failed: expected list, got string("wat?")`)

	_, err = readInterfaces(twoDotOh, []map[string]interface{}{
		{
			"wat": "?",
		},
	})
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err, gc.ErrorMatches, `interface 0: interface 2.0 schema check failed: .*`)
}

func (s *interfaceSuite) checkInterface(c *gc.C, iface *interface_) {
	c.Check(iface.ID(), gc.Equals, 40)
	c.Check(iface.Name(), gc.Equals, "eth0")
	c.Check(iface.Type(), gc.Equals, "physical")
	c.Check(iface.Enabled(), jc.IsTrue)
	c.Check(iface.Tags(), jc.DeepEquals, []string{"foo", "bar"})

	c.Check(iface.MACAddress(), gc.Equals, "52:54:00:c9:6a:45")
	c.Check(iface.EffectiveMTU(), gc.Equals, 1500)

	c.Check(iface.Parents(), jc.DeepEquals, []string{"bond0"})
	c.Check(iface.Children(), jc.DeepEquals, []string{"eth0.1", "eth0.2"})

	vlan := iface.VLAN()
	c.Assert(vlan, gc.NotNil)
	c.Check(vlan.Name(), gc.Equals, "untagged")

	links := iface.Links()
	c.Assert(links, gc.HasLen, 1)
	c.Check(links[0].ID(), gc.Equals, 69)
}

func (s *interfaceSuite) TestReadInterfaces(c *gc.C) {
	interfaces, err := readInterfaces(twoDotOh, parseJSON(c, interfacesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(interfaces, gc.HasLen, 1)
	s.checkInterface(c, interfaces[0])
}

func (s *interfaceSuite) TestReadInterface(c *gc.C) {
	result, err := readInterface(twoDotOh, parseJSON(c, interfaceResponse))
	c.Assert(err, jc.ErrorIsNil)
	s.checkInterface(c, result)
}

func (*interfaceSuite) TestLowVersion(c *gc.C) {
	_, err := readInterfaces(version.MustParse("1.9.0"), parseJSON(c, interfacesResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
	c.Assert(err.Error(), gc.Equals, `no interface read func for version 1.9.0`)

	_, err = readInterface(version.MustParse("1.9.0"), parseJSON(c, interfaceResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
	c.Assert(err.Error(), gc.Equals, `no interface read func for version 1.9.0`)
}

func (*interfaceSuite) TestHighVersion(c *gc.C) {
	read, err := readInterfaces(version.MustParse("2.1.9"), parseJSON(c, interfacesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(read, gc.HasLen, 1)
	_, err = readInterface(version.MustParse("2.1.9"), parseJSON(c, interfaceResponse))
	c.Assert(err, jc.ErrorIsNil)
}

func (s *interfaceSuite) getServerAndNewInterface(c *gc.C) (*SimpleTestServer, *interface_) {
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)
	devices, err := controller.Devices(DevicesArgs{})
	c.Assert(err, jc.ErrorIsNil)
	device := devices[0].(*device)
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusOK, interfaceResponse)
	iface, err := device.CreateInterface(minimalCreateInterfaceArgs())
	c.Assert(err, jc.ErrorIsNil)
	return server, iface.(*interface_)
}

func (s *interfaceSuite) TestDelete(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	// Successful delete is 204 - StatusNoContent - We hope, would be consistent
	// with device deletions.
	server.AddDeleteResponse(iface.resourceURI, http.StatusNoContent, "")
	err := iface.Delete()
	c.Assert(err, jc.ErrorIsNil)
}

func (s *interfaceSuite) TestDelete404(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	// No path, so 404
	err := iface.Delete()
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (s *interfaceSuite) TestDeleteForbidden(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddDeleteResponse(iface.resourceURI, http.StatusForbidden, "")
	err := iface.Delete()
	c.Assert(err, jc.Satisfies, IsPermissionError)
}

func (s *interfaceSuite) TestDeleteUnknown(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddDeleteResponse(iface.resourceURI, http.StatusConflict, "")
	err := iface.Delete()
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
}

type fakeSubnet struct {
	Subnet
	id int
}

func (f *fakeSubnet) ID() int {
	return f.id
}

func (s *interfaceSuite) TestLinkSubnetArgs(c *gc.C) {
	for i, test := range []struct {
		args    LinkSubnetArgs
		errText string
	}{{
		errText: "missing Mode not valid",
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeDHCP},
		errText: "missing Subnet not valid",
	}, {
		args:    LinkSubnetArgs{Mode: InterfaceLinkMode("foo")},
		errText: `unknown Mode value ("foo") not valid`,
	}, {
		args: LinkSubnetArgs{Mode: LinkModeDHCP, Subnet: &fakeSubnet{}},
	}, {
		args: LinkSubnetArgs{Mode: LinkModeStatic, Subnet: &fakeSubnet{}},
	}, {
		args: LinkSubnetArgs{Mode: LinkModeLinkUp, Subnet: &fakeSubnet{}},
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeDHCP, Subnet: &fakeSubnet{}, IPAddress: "10.10.10.10"},
		errText: `setting IP Address when Mode is not LinkModeStatic not valid`,
	}, {
		args: LinkSubnetArgs{Mode: LinkModeStatic, Subnet: &fakeSubnet{}, IPAddress: "10.10.10.10"},
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeLinkUp, Subnet: &fakeSubnet{}, IPAddress: "10.10.10.10"},
		errText: `setting IP Address when Mode is not LinkModeStatic not valid`,
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeDHCP, Subnet: &fakeSubnet{}, DefaultGateway: true},
		errText: `specifying DefaultGateway for Mode "DHCP" not valid`,
	}, {
		args: LinkSubnetArgs{Mode: LinkModeStatic, Subnet: &fakeSubnet{}, DefaultGateway: true},
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeLinkUp, Subnet: &fakeSubnet{}, DefaultGateway: true},
		errText: `specifying DefaultGateway for Mode "LINK_UP" not valid`,
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

func (s *interfaceSuite) TestLinkSubnetValidates(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	err := iface.LinkSubnet(LinkSubnetArgs{})
	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err.Error(), gc.Equals, "missing Mode not valid")
}

func (s *interfaceSuite) TestLinkSubnetGood(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	// The changed information is there just for the test to show that the response
	// is parsed and the interface updated
	response := updateJSONMap(c, interfaceResponse, map[string]interface{}{
		"name": "eth42",
	})
	server.AddPostResponse(iface.resourceURI+"?op=link_subnet", http.StatusOK, response)
	args := LinkSubnetArgs{
		Mode:           LinkModeStatic,
		Subnet:         &fakeSubnet{id: 42},
		IPAddress:      "10.10.10.10",
		DefaultGateway: true,
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.ErrorIsNil)
	c.Check(iface.Name(), gc.Equals, "eth42")

	request := server.LastRequest()
	form := request.PostForm
	c.Assert(form.Get("mode"), gc.Equals, "STATIC")
	c.Assert(form.Get("subnet"), gc.Equals, "42")
	c.Assert(form.Get("ip_address"), gc.Equals, "10.10.10.10")
	c.Assert(form.Get("default_gateway"), gc.Equals, "true")
}

func (s *interfaceSuite) TestLinkSubnetMissing(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &fakeSubnet{id: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, IsBadRequestError)
}

func (s *interfaceSuite) TestLinkSubnetForbidden(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.resourceURI+"?op=link_subnet", http.StatusForbidden, "bad user")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &fakeSubnet{id: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, IsPermissionError)
	c.Check(err.Error(), gc.Equals, "bad user")
}

func (s *interfaceSuite) TestLinkSubnetNoAddressesAvailable(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.resourceURI+"?op=link_subnet", http.StatusServiceUnavailable, "no addresses")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &fakeSubnet{id: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, IsCannotCompleteError)
	c.Check(err.Error(), gc.Equals, "no addresses")
}

func (s *interfaceSuite) TestLinkSubnetUnknown(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.resourceURI+"?op=link_subnet", http.StatusMethodNotAllowed, "wat?")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &fakeSubnet{id: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

const (
	interfacesResponse = "[" + interfaceResponse + "]"
	interfaceResponse  = `
{
    "effective_mtu": 1500,
    "mac_address": "52:54:00:c9:6a:45",
    "children": ["eth0.1", "eth0.2"],
    "discovered": [],
    "params": "some params",
    "vlan": {
        "resource_uri": "/MAAS/api/2.0/vlans/1/",
        "id": 1,
        "secondary_rack": null,
        "mtu": 1500,
        "primary_rack": "4y3h7n",
        "name": "untagged",
        "fabric": "fabric-0",
        "dhcp_on": true,
        "vid": 0
    },
    "name": "eth0",
    "enabled": true,
    "parents": ["bond0"],
    "id": 40,
    "type": "physical",
    "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/interfaces/40/",
    "tags": ["foo", "bar"],
    "links": [
        {
            "id": 69,
            "mode": "auto",
            "subnet": {
                "resource_uri": "/MAAS/api/2.0/subnets/1/",
                "id": 1,
                "rdns_mode": 2,
                "vlan": {
                    "resource_uri": "/MAAS/api/2.0/vlans/1/",
                    "id": 1,
                    "secondary_rack": null,
                    "mtu": 1500,
                    "primary_rack": "4y3h7n",
                    "name": "untagged",
                    "fabric": "fabric-0",
                    "dhcp_on": true,
                    "vid": 0
                },
                "dns_servers": [],
                "space": "space-0",
                "name": "192.168.100.0/24",
                "gateway_ip": "192.168.100.1",
                "cidr": "192.168.100.0/24"
            }
        }
    ]
}
`
)
