// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type blockdeviceSuite struct{}

var _ = gc.Suite(&blockdeviceSuite{})

func (*blockdeviceSuite) TestReadBlockDevicesBadSchema(c *gc.C) {
	_, err := readBlockDevices(twoDotOh, "wat?")
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `blockdevice base schema check failed: expected list, got string("wat?")`)
}

func (*blockdeviceSuite) TestReadBlockDevices(c *gc.C) {
	blockdevices, err := readBlockDevices(twoDotOh, parseJSON(c, blockdevicesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(blockdevices, gc.HasLen, 1)
	blockdevice := blockdevices[0]

	c.Check(blockdevice.ID(), gc.Equals, 34)
	c.Check(blockdevice.Name(), gc.Equals, "sda")
	c.Check(blockdevice.Model(), gc.Equals, "QEMU HARDDISK")
	c.Check(blockdevice.Path(), gc.Equals, "/dev/disk/by-dname/sda")
	c.Check(blockdevice.IDPath(), gc.Equals, "/dev/disk/by-id/ata-QEMU_HARDDISK_QM00001")
	c.Check(blockdevice.UUID(), gc.Equals, "6199b7c9-b66f-40f6-a238-a938a58a0adf")
	c.Check(blockdevice.UsedFor(), gc.Equals, "MBR partitioned with 1 partition")
	c.Check(blockdevice.Tags(), jc.DeepEquals, []string{"rotary"})
	c.Check(blockdevice.BlockSize(), gc.Equals, uint64(4096))
	c.Check(blockdevice.UsedSize(), gc.Equals, uint64(8586788864))
	c.Check(blockdevice.Size(), gc.Equals, uint64(8589934592))

	partitions := blockdevice.Partitions()
	c.Assert(partitions, gc.HasLen, 1)
	partition := partitions[0]
	c.Check(partition.ID(), gc.Equals, 1)
	c.Check(partition.UsedFor(), gc.Equals, "ext4 formatted filesystem mounted at /")

	fs := blockdevice.FileSystem()
	c.Assert(fs, gc.NotNil)
	c.Assert(fs.Type(), gc.Equals, "ext4")
	c.Assert(fs.MountPoint(), gc.Equals, "/srv")
}

func (*blockdeviceSuite) TestReadBlockDevicesWithNulls(c *gc.C) {
	blockdevices, err := readBlockDevices(twoDotOh, parseJSON(c, blockdevicesWithNullsResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(blockdevices, gc.HasLen, 1)
	blockdevice := blockdevices[0]

	c.Check(blockdevice.Model(), gc.Equals, "")
	c.Check(blockdevice.IDPath(), gc.Equals, "")
	c.Check(blockdevice.FileSystem(), gc.IsNil)
}

func (*blockdeviceSuite) TestLowVersion(c *gc.C) {
	_, err := readBlockDevices(version.MustParse("1.9.0"), parseJSON(c, blockdevicesResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (*blockdeviceSuite) TestHighVersion(c *gc.C) {
	blockdevices, err := readBlockDevices(version.MustParse("2.1.9"), parseJSON(c, blockdevicesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(blockdevices, gc.HasLen, 1)
}

var blockdevicesResponse = `
[
    {
        "path": "/dev/disk/by-dname/sda",
        "name": "sda",
        "used_for": "MBR partitioned with 1 partition",
        "partitions": [
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
                "size": 8581545984
            }
        ],
        "filesystem": {
            "fstype": "ext4",
            "mount_point": "/srv",
            "label": "root",
            "mount_options": null,
            "uuid": "fcd7745e-f1b5-4f5d-9575-9b0bb796b752"
        },
        "id_path": "/dev/disk/by-id/ata-QEMU_HARDDISK_QM00001",
        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/",
        "id": 34,
        "serial": "QM00001",
        "type": "physical",
        "block_size": 4096,
        "used_size": 8586788864,
        "available_size": 0,
        "partition_table_type": "MBR",
        "uuid": "6199b7c9-b66f-40f6-a238-a938a58a0adf",
        "size": 8589934592,
        "model": "QEMU HARDDISK",
        "tags": [
            "rotary"
        ]
    }
]
`

var blockdevicesWithNullsResponse = `
[
    {
        "path": "/dev/disk/by-dname/sda",
        "name": "sda",
        "used_for": "MBR partitioned with 1 partition",
        "partitions": [],
        "filesystem": null,
        "id_path": null,
        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/",
        "id": 34,
        "serial": null,
        "type": "physical",
        "block_size": 4096,
        "used_size": 8586788864,
        "available_size": 0,
        "partition_table_type": null,
        "uuid": null,
        "size": 8589934592,
        "model": null,
        "tags": []
    }
]
`
