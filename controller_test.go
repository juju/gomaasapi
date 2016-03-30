// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"net/http"

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
	server.AddResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddResponse("/api/2.0/zones/", http.StatusOK, zoneResponse)
	server.AddResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
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

func (s *controllerSuite) TestZones(c *gc.C) {
	controller := s.getController(c)
	zones, err := controller.Zones()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zones, gc.HasLen, 2)
}

func (s *controllerSuite) TestMachines(c *gc.C) {
	controller := s.getController(c)
	machines, err := controller.Machines(MachinesParams{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(machines, gc.HasLen, 3)
}

var versionResponse = `{"version": "unknown", "subversion": "", "capabilities": ["networks-management", "static-ipaddresses", "ipv6-deployment-ubuntu", "devices-management", "storage-deployment-ubuntu", "network-deployment-ubuntu"]}`
