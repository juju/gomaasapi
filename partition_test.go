// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type partitionSuite struct{}

var _ = gc.Suite(&partitionSuite{})

func (*partitionSuite) TestTypePartition(c *gc.C) {
	var empty partition
	c.Assert(empty.Type() == "partition", jc.IsTrue)
}

func (*partitionSuite) TestNilFileSystem(c *gc.C) {
	var empty partition
	c.Assert(empty.FileSystem() == nil, jc.IsTrue)
}

func (*partitionSuite) TestReadPartitionsBadSchema(c *gc.C) {
	_, err := readPartitions(twoDotOh, "wat?")
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `partition base schema check failed: expected list, got string("wat?")`)
}

func (*partitionSuite) TestReadPartitions(c *gc.C) {
	partitions, err := readPartitions(twoDotOh, parseJSON(c, partitionsResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(partitions, gc.HasLen, 1)
	partition := partitions[0]

	c.Check(partition.Type(), gc.Equals, "partition")
	c.Check(partition.ID(), gc.Equals, 1)
	c.Check(partition.Path(), gc.Equals, "/dev/disk/by-dname/sda-part1")
	c.Check(partition.UUID(), gc.Equals, "6199b7c9-b66f-40f6-a238-a938a58a0adf")
	c.Check(partition.UsedFor(), gc.Equals, "ext4 formatted filesystem mounted at /")
	c.Check(partition.Size(), gc.Equals, uint64(8581545984))
	c.Check(partition.Tags(), gc.DeepEquals, []string{"ssd-part", "osd-part"})

	fs := partition.FileSystem()
	c.Assert(fs, gc.NotNil)
	c.Assert(fs.Type(), gc.Equals, "ext4")
	c.Assert(fs.MountPoint(), gc.Equals, "/")
}

func (*partitionSuite) TestReadPartitionsNilUUID(c *gc.C) {
	json := parseJSON(c, partitionsResponse)
	json.([]interface{})[0].(map[string]interface{})["uuid"] = nil
	partitions, err := readPartitions(twoDotOh, json)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(partitions, gc.HasLen, 1)
	partition := partitions[0]
	c.Check(partition.UUID(), gc.Equals, "")
}

func (*partitionSuite) TestLowVersion(c *gc.C) {
	_, err := readPartitions(version.MustParse("1.9.0"), parseJSON(c, partitionsResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (*partitionSuite) TestHighVersion(c *gc.C) {
	partitions, err := readPartitions(version.MustParse("2.1.9"), parseJSON(c, partitionsResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(partitions, gc.HasLen, 1)
}

var partitionsResponse = `
[
    {
        "bootable": false,
        "id": 1,
        "path": "/dev/disk/by-dname/sda-part1",
        "filesystem": {
            "fstype": "ext4",
            "mount_point": "/",
            "label": "root",
            "mount_options": null,
            "uuid": "fcd7745e-f1b5-4f5d-9575-9b0bb796b752"
        },
        "type": "partition",
        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/partition/1",
        "uuid": "6199b7c9-b66f-40f6-a238-a938a58a0adf",
        "used_for": "ext4 formatted filesystem mounted at /",
		"size": 8581545984,
		"tags": ["ssd-part", "osd-part"]
    }
]
`
