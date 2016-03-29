// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"net/http"

	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type versionSuite struct{}

var _ = gc.Suite(&versionSuite{})

func (*versionSuite) TestSupportedVersions(c *gc.C) {
	for _, apiVersion := range supportedAPIVersions {
		_, _, err := version.ParseMajorMinor(apiVersion)
		c.Check(err, jc.ErrorIsNil)
	}
}

type controllerSuite struct{}

var _ = gc.Suite(&controllerSuite{})

func (*controllerSuite) TestNewController(c *gc.C) {
	server := NewSimpleServer()
	server.addResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	c.Assert(err, jc.ErrorIsNil)

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

var versionResponse = `{"version": "unknown", "subversion": "", "capabilities": ["networks-management", "static-ipaddresses", "ipv6-deployment-ubuntu", "devices-management", "storage-deployment-ubuntu", "network-deployment-ubuntu"]}`
