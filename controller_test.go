// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"net/http"

	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
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
	testing.CleanupSuite
	server *SimpleTestServer
}

var _ = gc.Suite(&controllerSuite{})

func (s *controllerSuite) SetUpTest(c *gc.C) {
	s.CleanupSuite.SetUpTest(c)

	server := NewSimpleServer()
	server.AddGetResponse("/api/2.0/boot-resources/", http.StatusOK, bootResourcesResponse)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)
	server.AddGetResponse("/api/2.0/fabrics/", http.StatusOK, fabricResponse)
	server.AddGetResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
	server.AddGetResponse("/api/2.0/machines/?hostname=untasted-markita", http.StatusOK, "["+machineResponse+"]")
	server.AddGetResponse("/api/2.0/spaces/", http.StatusOK, spacesResponse)
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

func (s *controllerSuite) TestDevicessArgs(c *gc.C) {
	controller := s.getController(c)
	// This will fail with a 404 due to the test server not having something  at
	// that address, but we don't care, all we want to do is capture the request
	// and make sure that all the values were set.
	controller.Devices(DevicesArgs{
		Hostname:     "untasted-markita",
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
	s.server.AddPostResponse("/api/2.0/devices/?op=create", http.StatusOK, deviceResponse)
	controller := s.getController(c)
	device, err := controller.CreateDevice(CreateDeviceArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(device.SystemID(), gc.Equals, "4y3ha8")
}

func (s *controllerSuite) TestCreateDeviceMissingAddress(c *gc.C) {
	controller := s.getController(c)
	_, err := controller.CreateDevice(CreateDeviceArgs{})
	c.Assert(err, jc.Satisfies, IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "at least one MAC address must be specified")
}

func (s *controllerSuite) TestCreateDeviceBadRequest(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/devices/?op=create", http.StatusBadRequest, "some error")
	controller := s.getController(c)
	_, err := controller.CreateDevice(CreateDeviceArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	c.Assert(err, jc.Satisfies, IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "some error")
}

func (s *controllerSuite) TestCreateDeviceArgs(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/devices/?op=create", http.StatusOK, deviceResponse)
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

func (s *controllerSuite) TestAllocateMachine(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusOK, machineResponse)
	controller := s.getController(c)
	machine, err := controller.AllocateMachine(AllocateMachineArgs{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machine.SystemID(), gc.Equals, "4y3ha3")
}

func (s *controllerSuite) TestAllocateMachineArgs(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusOK, machineResponse)
	controller := s.getController(c)
	// Create an arg structure that sets all the values.
	args := AllocateMachineArgs{
		Hostname:     "foobar",
		Architecture: "amd64",
		MinCPUCount:  42,
		MinMemory:    20000,
		Tags:         []string{"good"},
		NotTags:      []string{"bad"},
		Networks:     []string{"fast"},
		NotNetworks:  []string{"slow"},
		Zone:         "magic",
		NotInZone:    []string{"not-magic"},
		AgentName:    "agent 42",
		Comment:      "testing",
		DryRun:       true,
	}
	_, err := controller.AllocateMachine(args)
	c.Assert(err, jc.ErrorIsNil)

	request := s.server.LastRequest()
	// There should be one entry in the form values for each of the args.
	c.Assert(request.PostForm, gc.HasLen, 13)
}

func (s *controllerSuite) TestAllocateMachineNoMatch(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusConflict, "boo")
	controller := s.getController(c)
	_, err := controller.AllocateMachine(AllocateMachineArgs{})
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (s *controllerSuite) TestAllocateMachineUnexpected(c *gc.C) {
	s.server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusBadRequest, "boo")
	controller := s.getController(c)
	_, err := controller.AllocateMachine(AllocateMachineArgs{})
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

var versionResponse = `{"version": "unknown", "subversion": "", "capabilities": ["networks-management", "static-ipaddresses", "ipv6-deployment-ubuntu", "devices-management", "storage-deployment-ubuntu", "network-deployment-ubuntu"]}`

type cleanup interface {
	AddCleanup(testing.CleanupFunc)
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
