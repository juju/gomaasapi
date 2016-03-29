// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
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
