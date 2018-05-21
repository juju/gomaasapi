// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type domainSuite struct{}

var _ = gc.Suite(&domainSuite{})

func (*domainSuite) TestReadDomainsBadSchema(c *gc.C) {
	_, err := readDomains(twoDotOh, "something")
	c.Assert(err.Error(), gc.Equals, `domain base schema check failed: expected list, got string("something")`)
}

func (*domainSuite) TestReadDomains(c *gc.C) {
	domains, err := readDomains(twoDotOh, parseJSON(c, domainResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(domains, gc.HasLen, 2)
	c.Assert(domains[0].Name(), gc.Equals, "maas")
	c.Assert(domains[1].Name(), gc.Equals, "anotherDomain.com")
}

var domainResponse = `
[
    {
        "authoritative": "true",
        "resource_uri": "/MAAS/api/2.0/domains/0/",
        "name": "maas",
        "id": 0,
        "ttl": null,
        "resource_record_count": 3
    }, {
        "authoritative": "true",
        "resource_uri": "/MAAS/api/2.0/domains/1/",
        "name": "anotherDomain.com",
        "id": 1,
        "ttl": 10,
        "resource_record_count": 3
    }
]
`
