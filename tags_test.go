package gomaasapi

import (
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type tagSuite struct {
	testing.LoggingCleanupSuite
}

var _ = gc.Suite(&tagSuite{})

func (*tagSuite) TestReadTags(c *gc.C) {
	tags, err := readTags(twoDotOh, parseJSON(c, tagsResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(tags, gc.HasLen, 3)

	tag := tags[0]

	c.Check(tag.Name(), gc.Equals, "virtual")
	c.Check(tag.Comment(), gc.Equals, "virtual machines")
	c.Check(tag.Definition(), gc.Equals, "tag for machines that are virtual")
	c.Check(tag.KernelOpts(), gc.Equals, "nvme_core")
}

var tagsResponse = `[
	{
		"resource_uri": "/2.0/tags/virtual",
		"name": "virtual",
		"comment": "virtual machines",
		"definition": "tag for machines that are virtual",
		"kernel_opts": "nvme_core"
	},
	{
		"resource_uri": "/2.0/tags/physical",
		"name": "physical",
		"comment": "physical machines",
		"definition": "tag for machines that are physical",
		"kernel_opts": ""
	},
	{
		"resource_uri": "/2.0/tags/r-pi",
		"name": "r-pi",
		"comment": "raspberry pis'",
		"definition": "tag for machines that are raspberry pis",
		"kernel_opts": ""
	}
]`
