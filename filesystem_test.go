// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type filesystemSuite struct{}

var _ = gc.Suite(&filesystemSuite{})

func (*filesystemSuite) TestParse2_0(c *gc.C) {
	source := map[string]interface{}{
		"fstype":      "ext4",
		"mount_point": "/",
		"label":       "root",
		"uuid":        "fake-uuid",
	}
	fs, err := filesystem2_0(source)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(fs.Type(), gc.Equals, "ext4")
	c.Check(fs.MountPoint(), gc.Equals, "/")
	c.Check(fs.Label(), gc.Equals, "root")
	c.Check(fs.UUID(), gc.Equals, "fake-uuid")
}

func (*filesystemSuite) TestParse2_Defaults(c *gc.C) {
	source := map[string]interface{}{
		"fstype":      "ext4",
		"mount_point": nil,
		"label":       nil,
		"uuid":        "fake-uuid",
	}
	fs, err := filesystem2_0(source)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(fs.Type(), gc.Equals, "ext4")
	c.Check(fs.MountPoint(), gc.Equals, "")
	c.Check(fs.Label(), gc.Equals, "")
	c.Check(fs.UUID(), gc.Equals, "fake-uuid")
}

func (*filesystemSuite) TestParse2_0BadSchema(c *gc.C) {
	source := map[string]interface{}{
		"mount_point": "/",
		"label":       "root",
		"uuid":        "fake-uuid",
	}
	_, err := filesystem2_0(source)
	c.Assert(err, jc.Satisfies, IsDeserializationError)
}
