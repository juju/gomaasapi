// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type bootResourceSuite struct{}

var _ = gc.Suite(&bootResourceSuite{})

func (*bootResourceSuite) TestReadBootResourcesBadSchema(c *gc.C) {
	_, err := readBootResources(twoDotOh, "wat?")
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `boot resource base schema check failed: expected list, got string("wat?")`)
}

func (*bootResourceSuite) TestReadBootResources(c *gc.C) {
	bootResources, err := readBootResources(twoDotOh, parseJSON(c, bootResourcesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(bootResources, gc.HasLen, 6)
	trusty := bootResources[0]

	subarches := set.NewStrings("generic", "hwe-p", "hwe-q", "hwe-r", "hwe-s", "hwe-t")
	c.Assert(trusty.ID(), gc.Equals, 5)
	c.Assert(trusty.Name(), gc.Equals, "ubuntu/trusty")
	c.Assert(trusty.Type(), gc.Equals, "Synced")
	c.Assert(trusty.Architecture(), gc.Equals, "amd64/hwe-t")
	c.Assert(trusty.SubArchitectures(), jc.DeepEquals, subarches)
	c.Assert(trusty.KernelFlavor(), gc.Equals, "generic")

	centos := bootResources[5]
	c.Assert(centos.KernelFlavor(), gc.Equals, "")
}

func (*bootResourceSuite) TestLowVersion(c *gc.C) {
	_, err := readBootResources(version.MustParse("1.9.0"), parseJSON(c, bootResourcesResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (*bootResourceSuite) TestHighVersion(c *gc.C) {
	bootResources, err := readBootResources(version.MustParse("2.1.9"), parseJSON(c, bootResourcesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(bootResources, gc.HasLen, 6)
}

var bootResourcesResponse = `
[
    {
        "architecture": "amd64/hwe-t",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t",
        "kflavor": "generic",
        "name": "ubuntu/trusty",
        "id": 5,
        "resource_uri": "/MAAS/api/2.0/boot-resources/5/"
    },
    {
        "architecture": "amd64/hwe-u",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t,hwe-u",
        "kflavor": "generic",
        "name": "ubuntu/trusty",
        "id": 1,
        "resource_uri": "/MAAS/api/2.0/boot-resources/1/"
    },
    {
        "architecture": "amd64/hwe-v",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t,hwe-u,hwe-v",
        "kflavor": "generic",
        "name": "ubuntu/trusty",
        "id": 3,
        "resource_uri": "/MAAS/api/2.0/boot-resources/3/"
    },
    {
        "architecture": "amd64/hwe-w",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t,hwe-u,hwe-v,hwe-w",
        "kflavor": "generic",
        "name": "ubuntu/trusty",
        "id": 4,
        "resource_uri": "/MAAS/api/2.0/boot-resources/4/"
    },
    {
        "architecture": "amd64/hwe-x",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t,hwe-u,hwe-v,hwe-w,hwe-x",
        "kflavor": "generic",
        "name": "ubuntu/xenial",
        "id": 2,
        "resource_uri": "/MAAS/api/2.0/boot-resources/2/"
    },
    {
        "type": "Generated",
        "architecture": "amd64/generic",
        "resource_uri": "/MAAS/api/2.0/boot-resources/6/",
        "name": "centos/centos7",
        "title": "",
        "subarches": "generic",
        "id": 6
    }
]
`
