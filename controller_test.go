// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/juju/collections/set"
	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type versionSuite struct {
}

var _ = gc.Suite(&versionSuite{})

func (*versionSuite) TestSupportedVersions(c *gc.C) {
	for _, apiVersion := range supportedAPIVersions {
		_, _, err := version.ParseMajorMinor(apiVersion)
		c.Check(err, jc.ErrorIsNil)
	}
}

type controllerSuite struct {
	testing.LoggingCleanupSuite
	server *SimpleTestServer
}

var _ = gc.Suite(&controllerSuite{})

func (s *controllerSuite) SetUpTest(c *gc.C) {
	s.LoggingCleanupSuite.SetUpTest(c)
	loggo.GetLogger("").SetLogLevel(loggo.TRACE)

	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/boot-resources/", http.StatusOK, bootResourcesResponse)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)
	server.AddGetResponse("/api/2.0/fabrics/", http.StatusOK, fabricResponse)
	server.AddGetResponse("/api/2.0/files/", http.StatusOK, filesResponse)
	server.AddGetResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
	server.AddGetResponse("/api/2.0/machines/?hostname=untasted-markita", http.StatusOK, "["+machineResponse+"]")
	server.AddGetResponse("/api/2.0/spaces/", http.StatusOK, spacesResponse)
	server.AddGetResponse("/api/2.0/static-routes/", http.StatusOK, staticRoutesResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/zones/", http.StatusOK, zoneResponse)
	server.Start()
	s.AddCleanup(func(*gc.C) { server.Close() })
	s.server = server
}

func (s *controllerSuite) getController(c *gc.C) Controller {
	controller, err := NewController(ControllerArgs{
		BaseURL: s.server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(err, jc.ErrorIsNil)
	return controller
}

func (s *controllerSuite) TestNewController(c *gc.C) {
	controller := s.getController(c)

	expectedCapabilities := set.NewStrings(
		NetworksManagement,
		StaticIPAddresses,
		IPv6DeploymentUbuntu,
		DevicesManagement,
		StorageDeploymentUbuntu,
		NetworkDeploymentUbuntu,
	)

	capabilities := controller.Capabilities()
	c.Assert(capabilities.Difference(expectedCapabilities), gc.HasLen, 0)
	c.Assert(expectedCapabilities.Difference(capabilities), gc.HasLen, 0)
}

func (s *controllerSuite) TestNewControllerBadAPIKeyFormat(c *gc.C) {
	server := NewSimpleServer()
	server.Start()
	defer server.Close()
	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "invalid",
	})
	c.Assert(err, jc.Satisfies, errors.IsNotValid)
}

func (s *controllerSuite) TestNewControllerNoSupport(c *gc.C) {
	server := NewSimpleServer()
	server.Start()
	defer server.Close()
	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (s *controllerSuite) TestNewControllerBadCreds(c *gc.C) {
	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusUnauthorized, "naughty")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	defer server.Close()
	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(err, jc.Satisfies, IsPermissionError)
}

func (s *controllerSuite) TestNewControllerUnexpected(c *gc.C) {
	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusConflict, "naughty")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	defer server.Close()
	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
}

func (s *controllerSuite) TestNewControllerKnownVersion(c *gc.C) {
	// Using a server URL including the version should work.
	officialController, err := NewController(ControllerArgs{
		BaseURL: s.server.URL + "/api/2.0/",
		APIKey:  "fake:as:key",
	})
	c.Assert(err, jc.ErrorIsNil)
	rawController, ok := officialController.(*controller)
	c.Assert(ok, jc.IsTrue)
	c.Assert(rawController.apiVersion, gc.Equals, version.Number{
		Major: 2,
		Minor: 0,
	})
}

func (s *controllerSuite) TestNewControllerUnsupportedVersionSpecified(c *gc.C) {
	// Ensure the server would actually respond to the version if it
	// was asked.
	s.server.AddGetResponse("/api/3.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	s.server.AddGetResponse("/api/3.0/version/", http.StatusOK, versionResponse)
	// Using a server URL including a version that isn't in the known
	// set should be denied.
	controller, err := NewController(ControllerArgs{
		BaseURL: s.server.URL + "/api/3.0/",
		APIKey:  "fake:as:key",
	})
	c.Assert(controller, gc.IsNil)
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (s *controllerSuite) TestNewControllerNotHidingErrors(c *gc.C) {
	// We should only treat 404 and 410 as "this version isn't
	// supported". Other errors should be returned up the stack
	// unchanged, so we don't confuse transient network errors with
	// version mismatches. lp:1667095
	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "underwater woman")
	server.AddGetResponse("/api/2.0/version/", http.StatusInternalServerError, "kablooey")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(controller, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, `ServerError: 500 Internal Server Error \(kablooey\)`)
}

func (s *controllerSuite) TestNewController410(c *gc.C) {
	// We should only treat 404 and 410 as "this version isn't
	// supported". Other errors should be returned up the stack
	// unchanged, so we don't confuse transient network errors with
	// version mismatches. lp:1667095
	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "the answer to all your prayers")
	server.AddGetResponse("/api/2.0/version/", http.StatusGone, "cya")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(controller, gc.IsNil)
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (s *controllerSuite) TestNewController404(c *gc.C) {
	// We should only treat 404 and 410 as "this version isn't
	// supported". Other errors should be returned up the stack
	// unchanged, so we don't confuse transient network errors with
	// version mismatches. lp:1667095
	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "the answer to all your prayers")
	server.AddGetResponse("/api/2.0/version/", http.StatusNotFound, "huh?")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(controller, gc.IsNil)
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (s *controllerSuite) TestNewControllerWith194Bug(c *gc.C) {
	// 1.9.4 has a bug where if you ask for /api/2.0/version/ without
	// being logged in (rather than OAuth connection) it redirects you
	// to the login page. This is fixed in 1.9.5, but we should work
	// around it anyway. https://bugs.launchpad.net/maas/+bug/1583715
	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "the answer to all your prayers")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, "<html><head>")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(controller, gc.IsNil)
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (s *controllerSuite) TestBootResources(c *gc.C) {
	controller := s.getController(c)
	resources, err := controller.BootResources()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(resources, gc.HasLen, 5)
}

func (s *controllerSuite) TestDevices(c *gc.C) {
	controller := s.getController(c)
	devices, err := controller.Devices(DevicesArgs{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(devices, gc.HasLen, 1)
}

func (s *controllerSuite) TestDevicesArgs(c *gc.C) {
	controller := s.getController(c)
	// This will fail with a 404 due to the test server not having something  at
	// that address, but we don't care, all we want to do is capture the request
	// and make sure that all the values were set.
	controller.Devices(DevicesArgs{
		Hostname:     []string{"untasted-markita"},
		MACAddresses: []string{"something"},
		SystemIDs:    []string{"something-else"},
		Domain:       "magic",
		Zone:         "foo",
		AgentName:    "agent 42",
	})
	request := s.server.LastRequest()
	// There should be one entry in the form values for each of the args.
	c.Assert(request.URL.Query(), gc.HasLen, 6)
}

func (s *controllerSuite) TestCreateDevice(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, deviceResponse)
	controller := s.getController(c)
	device, err := controller.CreateDevice(CreateDeviceArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(device.SystemID(), gc.Equals, "4y3haf")
}

func (s *controllerSuite) TestCreateDeviceMissingAddress(c *gc.C) {
	controller := s.getController(c)
	_, err := controller.CreateDevice(CreateDeviceArgs{})
	c.Assert(err, jc.Satisfies, IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "at least one MAC address must be specified")
}

func (s *controllerSuite) TestCreateDeviceBadRequest(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/devices/?op=", http.StatusBadRequest, "some error")
	controller := s.getController(c)
	_, err := controller.CreateDevice(CreateDeviceArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	c.Assert(err, jc.Satisfies, IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "some error")
}

func (s *controllerSuite) TestCreateDeviceArgs(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, deviceResponse)
	controller := s.getController(c)
	// Create an arg structure that sets all the values.
	args := CreateDeviceArgs{
		Hostname:     "foobar",
		MACAddresses: []string{"an-address"},
		Domain:       "a domain",
		Parent:       "parent",
	}
	_, err := controller.CreateDevice(args)
	c.Assert(err, jc.ErrorIsNil)

	request := s.server.LastRequest()
	// There should be one entry in the form values for each of the args.
	c.Assert(request.PostForm, gc.HasLen, 4)
}

func (s *controllerSuite) TestFabrics(c *gc.C) {
	controller := s.getController(c)
	fabrics, err := controller.Fabrics()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(fabrics, gc.HasLen, 2)
}

func (s *controllerSuite) TestSpaces(c *gc.C) {
	controller := s.getController(c)
	spaces, err := controller.Spaces()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(spaces, gc.HasLen, 1)
}

func (s *controllerSuite) TestStaticRoutes(c *gc.C) {
	controller := s.getController(c)
	staticRoutes, err := controller.StaticRoutes()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(staticRoutes, gc.HasLen, 1)
}

func (s *controllerSuite) TestZones(c *gc.C) {
	controller := s.getController(c)
	zones, err := controller.Zones()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zones, gc.HasLen, 2)
}

func (s *controllerSuite) TestMachines(c *gc.C) {
	controller := s.getController(c)
	machines, err := controller.Machines(MachinesArgs{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machines, gc.HasLen, 3)
}

func (s *controllerSuite) TestMachinesFilter(c *gc.C) {
	controller := s.getController(c)
	machines, err := controller.Machines(MachinesArgs{
		Hostnames: []string{"untasted-markita"},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machines, gc.HasLen, 1)
	c.Assert(machines[0].Hostname(), gc.Equals, "untasted-markita")
}

func (s *controllerSuite) TestMachinesFilterWithOwnerData(c *gc.C) {
	controller := s.getController(c)
	machines, err := controller.Machines(MachinesArgs{
		Hostnames: []string{"untasted-markita"},
		OwnerData: map[string]string{
			"fez": "jim crawford",
		},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machines, gc.HasLen, 0)
}

func (s *controllerSuite) TestMachinesFilterWithOwnerData_MultipleMatches(c *gc.C) {
	controller := s.getController(c)
	machines, err := controller.Machines(MachinesArgs{
		OwnerData: map[string]string{
			"braid": "jonathan blow",
		},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machines, gc.HasLen, 2)
	c.Assert(machines[0].Hostname(), gc.Equals, "lowlier-glady")
	c.Assert(machines[1].Hostname(), gc.Equals, "icier-nina")
}

func (s *controllerSuite) TestMachinesFilterWithOwnerData_RequiresAllMatch(c *gc.C) {
	controller := s.getController(c)
	machines, err := controller.Machines(MachinesArgs{
		OwnerData: map[string]string{
			"braid":          "jonathan blow",
			"frog-fractions": "jim crawford",
		},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machines, gc.HasLen, 1)
	c.Assert(machines[0].Hostname(), gc.Equals, "lowlier-glady")
}

func (s *controllerSuite) TestMachinesArgs(c *gc.C) {
	controller := s.getController(c)
	// This will fail with a 404 due to the test server not having something  at
	// that address, but we don't care, all we want to do is capture the request
	// and make sure that all the values were set.
	controller.Machines(MachinesArgs{
		Hostnames:    []string{"untasted-markita"},
		MACAddresses: []string{"something"},
		SystemIDs:    []string{"something-else"},
		Domain:       "magic",
		Zone:         "foo",
		AgentName:    "agent 42",
	})
	request := s.server.LastRequest()
	// There should be one entry in the form values for each of the args.
	c.Assert(request.URL.Query(), gc.HasLen, 6)
}

func (s *controllerSuite) TestStorageSpec(c *gc.C) {
	for i, test := range []struct {
		spec StorageSpec
		err  string
		repr string
	}{{
		spec: StorageSpec{},
		err:  "Size value 0 not valid",
	}, {
		spec: StorageSpec{Size: -10},
		err:  "Size value -10 not valid",
	}, {
		spec: StorageSpec{Size: 200},
		repr: "200",
	}, {
		spec: StorageSpec{Label: "foo", Size: 200},
		repr: "foo:200",
	}, {
		spec: StorageSpec{Size: 200, Tags: []string{"foo", ""}},
		err:  "empty tag not valid",
	}, {
		spec: StorageSpec{Size: 200, Tags: []string{"foo"}},
		repr: "200(foo)",
	}, {
		spec: StorageSpec{Label: "omg", Size: 200, Tags: []string{"foo", "bar"}},
		repr: "omg:200(foo,bar)",
	}} {
		c.Logf("test %d", i)
		err := test.spec.Validate()
		if test.err == "" {
			c.Assert(err, jc.ErrorIsNil)
			c.Assert(test.spec.String(), gc.Equals, test.repr)
		} else {
			c.Assert(err, jc.Satisfies, errors.IsNotValid)
			c.Assert(err.Error(), gc.Equals, test.err)
		}
	}
}

func (s *controllerSuite) TestInterfaceSpec(c *gc.C) {
	for i, test := range []struct {
		spec InterfaceSpec
		err  string
		repr string
	}{{
		spec: InterfaceSpec{},
		err:  "missing Label not valid",
	}, {
		spec: InterfaceSpec{Label: "foo"},
		err:  "empty Space constraint not valid",
	}, {
		spec: InterfaceSpec{Label: "foo", Space: "magic"},
		repr: "foo:space=magic",
	}} {
		c.Logf("test %d", i)
		err := test.spec.Validate()
		if test.err == "" {
			c.Check(err, jc.ErrorIsNil)
			c.Check(test.spec.String(), gc.Equals, test.repr)
		} else {
			c.Check(err, jc.Satisfies, errors.IsNotValid)
			c.Check(err.Error(), gc.Equals, test.err)
		}
	}
}

func (s *controllerSuite) TestAllocateMachineArgs(c *gc.C) {
	for i, test := range []struct {
		args       AllocateMachineArgs
		err        string
		storage    string
		interfaces string
		notSubnets []string
	}{{
		args: AllocateMachineArgs{},
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{{}},
		},
		err: "Storage: Size value 0 not valid",
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{{Size: 200}, {Size: 400, Tags: []string{"ssd"}}},
		},
		storage: "200,400(ssd)",
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{
				{Label: "foo", Size: 200},
				{Label: "foo", Size: 400, Tags: []string{"ssd"}},
			},
		},
		err: `reusing storage label "foo" not valid`,
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{{}},
		},
		err: "Interfaces: missing Label not valid",
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{
				{Label: "foo", Space: "magic"},
				{Label: "bar", Space: "other"},
			},
		},
		interfaces: "foo:space=magic;bar:space=other",
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{
				{Label: "foo", Space: "magic"},
				{Label: "foo", Space: "other"},
			},
		},
		err: `reusing interface label "foo" not valid`,
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{""},
		},
		err: "empty NotSpace constraint not valid",
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{"foo"},
		},
		notSubnets: []string{"space:foo"},
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{"foo", "bar"},
		},
		notSubnets: []string{"space:foo", "space:bar"},
	}} {
		c.Logf("test %d", i)
		err := test.args.Validate()
		if test.err == "" {
			c.Check(err, jc.ErrorIsNil)
			c.Check(test.args.storage(), gc.Equals, test.storage)
			c.Check(test.args.interfaces(), gc.Equals, test.interfaces)
			c.Check(test.args.notSubnets(), jc.DeepEquals, test.notSubnets)
		} else {
			c.Check(err, jc.Satisfies, errors.IsNotValid)
			c.Check(err.Error(), gc.Equals, test.err)
		}
	}
}

type constraintMatchInfo map[string][]int

func (s *controllerSuite) addAllocateResponse(c *gc.C, status int, interfaceMatches, storageMatches constraintMatchInfo) {
	constraints := make(map[string]interface{})
	if interfaceMatches != nil {
		constraints["interfaces"] = interfaceMatches
	}
	if storageMatches != nil {
		constraints["storage"] = storageMatches
	}
	allocateJSON := updateJSONMap(c, machineResponse, map[string]interface{}{
		"constraints_by_type": constraints,
	})
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", status, allocateJSON)
}

func (s *controllerSuite) TestAllocateMachine(c *gc.C) {
	s.addAllocateResponse(c, http.StatusOK, nil, nil)
	controller := s.getController(c)
	machine, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machine.SystemID(), gc.Equals, "4y3ha3")
}

func (s *controllerSuite) TestAllocateMachineInterfacesMatch(c *gc.C) {
	s.addAllocateResponse(c, http.StatusOK, constraintMatchInfo{
		"database": []int{35, 99},
	}, nil)
	controller := s.getController(c)
	_, match, err := controller.AllocateMachine(AllocateMachineArgs{
		// This isn't actually used, but here to show how it should be used.
		Interfaces: []InterfaceSpec{{
			Label: "database",
			Space: "space-0",
		}},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(match.Interfaces, gc.HasLen, 1)
	ifaces := match.Interfaces["database"]
	c.Assert(ifaces, gc.HasLen, 2)
	c.Assert(ifaces[0].ID(), gc.Equals, 35)
	c.Assert(ifaces[1].ID(), gc.Equals, 99)
}

func (s *controllerSuite) TestAllocateMachineInterfacesMatchMissing(c *gc.C) {
	// This should never happen, but if it does it is a clear indication of a
	// bug somewhere.
	s.addAllocateResponse(c, http.StatusOK, constraintMatchInfo{
		"database": []int{40},
	}, nil)
	controller := s.getController(c)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{
		Interfaces: []InterfaceSpec{{
			Label: "database",
			Space: "space-0",
		}},
	})
	c.Assert(err, jc.Satisfies, IsDeserializationError)
}

func (s *controllerSuite) TestAllocateMachineStorageMatches(c *gc.C) {
	s.addAllocateResponse(c, http.StatusOK, nil, constraintMatchInfo{
		"root": []int{34, 98},
	})
	controller := s.getController(c)
	_, match, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Label: "root",
			Size:  50,
			Tags:  []string{"hefty", "tangy"},
		}},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(match.Storage, gc.HasLen, 1)
	storages := match.Storage["root"]
	c.Assert(storages, gc.HasLen, 2)
	c.Assert(storages[0].ID(), gc.Equals, 34)
	c.Assert(storages[1].ID(), gc.Equals, 98)
}

func (s *controllerSuite) TestAllocateMachineStorageLogicalMatches(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusOK, machineResponse)
	controller := s.getController(c)
	machine, matches, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{
			{
				Tags: []string{"raid0"},
			},
			{
				Tags: []string{"partition"},
			},
		},
	})
	c.Assert(err, jc.ErrorIsNil)
	var virtualDeviceID = 23
	var partitionID = 1

	//matches storage must contain the "raid0" virtual block device
	c.Assert(matches.Storage["0"][0], gc.Equals, machine.BlockDevice(virtualDeviceID))
	//matches storage must contain the partition from physical block device
	c.Assert(matches.Storage["1"][0], gc.Equals, machine.Partition(partitionID))
}

func (s *controllerSuite) TestAllocateMachineStorageMatchMissing(c *gc.C) {
	// This should never happen, but if it does it is a clear indication of a
	// bug somewhere.
	s.addAllocateResponse(c, http.StatusOK, nil, constraintMatchInfo{
		"root": []int{50},
	})
	controller := s.getController(c)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Label: "root",
			Size:  50,
			Tags:  []string{"hefty", "tangy"},
		}},
	})
	c.Assert(err, jc.Satisfies, IsDeserializationError)
}

func (s *controllerSuite) TestAllocateMachineArgsForm(c *gc.C) {
	s.addAllocateResponse(c, http.StatusOK, nil, nil)
	controller := s.getController(c)
	// Create an arg structure that sets all the values.
	args := AllocateMachineArgs{
		Hostname:     "foobar",
		SystemId:     "some_id",
		Architecture: "amd64",
		MinCPUCount:  42,
		MinMemory:    20000,
		Tags:         []string{"good"},
		NotTags:      []string{"bad"},
		Storage:      []StorageSpec{{Label: "root", Size: 200}},
		Interfaces:   []InterfaceSpec{{Label: "default", Space: "magic"}},
		NotSpace:     []string{"special"},
		Zone:         "magic",
		NotInZone:    []string{"not-magic"},
		AgentName:    "agent 42",
		Comment:      "testing",
		DryRun:       true,
	}
	_, _, err := controller.AllocateMachine(args)
	c.Assert(err, jc.ErrorIsNil)

	request := s.server.LastRequest()
	// There should be one entry in the form values for each of the args.
	form := request.PostForm
	c.Assert(form, gc.HasLen, 15)
	// Positive space check.
	c.Assert(form.Get("interfaces"), gc.Equals, "default:space=magic")
	// Negative space check.
	c.Assert(form.Get("not_subnets"), gc.Equals, "space:special")
}

func (s *controllerSuite) TestAllocateMachineNoMatch(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusConflict, "boo")
	controller := s.getController(c)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (s *controllerSuite) TestAllocateMachineUnexpected(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusBadRequest, "boo")
	controller := s.getController(c)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
}

func (s *controllerSuite) TestReleaseMachines(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusOK, "[]")
	controller := s.getController(c)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
		Comment:   "all good",
	})
	c.Assert(err, jc.ErrorIsNil)

	request := s.server.LastRequest()
	// There should be one entry in the form values for each of the args.
	c.Assert(request.PostForm["machines"], jc.SameContents, []string{"this", "that"})
	c.Assert(request.PostForm.Get("comment"), gc.Equals, "all good")
}

func (s *controllerSuite) TestReleaseMachinesBadRequest(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusBadRequest, "unknown machines")
	controller := s.getController(c)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "unknown machines")
}

func (s *controllerSuite) TestReleaseMachinesForbidden(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusForbidden, "bzzt denied")
	controller := s.getController(c)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, IsPermissionError)
	c.Assert(err.Error(), gc.Equals, "bzzt denied")
}

func (s *controllerSuite) TestReleaseMachinesConflict(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusConflict, "machine busy")
	controller := s.getController(c)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, IsCannotCompleteError)
	c.Assert(err.Error(), gc.Equals, "machine busy")
}

func (s *controllerSuite) TestReleaseMachinesUnexpected(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusBadGateway, "wat")
	controller := s.getController(c)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: ServerError: 502 Bad Gateway (wat)")
}

func (s *controllerSuite) TestFiles(c *gc.C) {
	controller := s.getController(c)
	files, err := controller.Files("")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(files, gc.HasLen, 2)

	file := files[0]
	c.Assert(file.Filename(), gc.Equals, "test")
	uri, err := url.Parse(file.AnonymousURL())
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(uri.Scheme, gc.Equals, "http")
	c.Assert(uri.RequestURI(), gc.Equals, "/MAAS/api/2.0/files/?op=get_by_key&key=3afba564-fb7d-11e5-932f-52540051bf22")
}

func (s *controllerSuite) TestGetFile(c *gc.C) {
	s.server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	controller := s.getController(c)
	file, err := controller.GetFile("testing")
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(file.Filename(), gc.Equals, "testing")
	uri, err := url.Parse(file.AnonymousURL())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(uri.Scheme, gc.Equals, "http")
	c.Assert(uri.RequestURI(), gc.Equals, "/MAAS/api/2.0/files/?op=get_by_key&key=88e64b76-fb82-11e5-932f-52540051bf22")
}

func (s *controllerSuite) TestGetFileMissing(c *gc.C) {
	controller := s.getController(c)
	_, err := controller.GetFile("missing")
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (s *controllerSuite) TestAddFileArgsValidate(c *gc.C) {
	reader := bytes.NewBufferString("test")
	for i, test := range []struct {
		args    AddFileArgs
		errText string
	}{{
		errText: "missing Filename not valid",
	}, {
		args:    AddFileArgs{Filename: "/foo"},
		errText: `paths in Filename "/foo" not valid`,
	}, {
		args:    AddFileArgs{Filename: "a/foo"},
		errText: `paths in Filename "a/foo" not valid`,
	}, {
		args:    AddFileArgs{Filename: "foo.txt"},
		errText: `missing Content or Reader not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Reader:   reader,
		},
		errText: `missing Length not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Reader:   reader,
			Length:   4,
		},
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Content:  []byte("foo"),
			Reader:   reader,
		},
		errText: `specifying Content and Reader not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Content:  []byte("foo"),
			Length:   20,
		},
		errText: `specifying Length and Content not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Content:  []byte("foo"),
		},
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

func (s *controllerSuite) TestAddFileValidates(c *gc.C) {
	controller := s.getController(c)
	err := controller.AddFile(AddFileArgs{})
	c.Assert(err, jc.Satisfies, errors.IsNotValid)
}

func (s *controllerSuite) assertFile(c *gc.C, request *http.Request, filename, content string) {
	form := request.Form
	c.Check(form.Get("filename"), gc.Equals, filename)
	fileHeader := request.MultipartForm.File["file"][0]
	f, err := fileHeader.Open()
	c.Assert(err, jc.ErrorIsNil)
	bytes, err := ioutil.ReadAll(f)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(string(bytes), gc.Equals, content)
}

func (s *controllerSuite) TestAddFileContent(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/files/?op=", http.StatusOK, "")
	controller := s.getController(c)
	err := controller.AddFile(AddFileArgs{
		Filename: "foo.txt",
		Content:  []byte("foo"),
	})
	c.Assert(err, jc.ErrorIsNil)

	request := s.server.LastRequest()
	s.assertFile(c, request, "foo.txt", "foo")
}

func (s *controllerSuite) TestAddFileReader(c *gc.C) {
	reader := bytes.NewBufferString("test\n extra over length ignored")
	s.server.AddPostResponse("/api/2.0/files/?op=", http.StatusOK, "")
	controller := s.getController(c)
	err := controller.AddFile(AddFileArgs{
		Filename: "foo.txt",
		Reader:   reader,
		Length:   5,
	})
	c.Assert(err, jc.ErrorIsNil)

	request := s.server.LastRequest()
	s.assertFile(c, request, "foo.txt", "test\n")
}

var versionResponse = `{"version": "unknown", "subversion": "", "capabilities": ["networks-management", "static-ipaddresses", "ipv6-deployment-ubuntu", "devices-management", "storage-deployment-ubuntu", "network-deployment-ubuntu"]}`

type cleanup interface {
	AddCleanup(func(*gc.C))
}

// createTestServerController creates a controller backed on to a test server
// that has sufficient knowledge of versions and users to be able to create a
// valid controller.
func createTestServerController(c *gc.C, suite cleanup) (*SimpleTestServer, Controller) {
	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	suite.AddCleanup(func(*gc.C) { server.Close() })

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(err, jc.ErrorIsNil)
	return server, controller
}
