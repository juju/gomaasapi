// Copyright 2026 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/json"

	. "gopkg.in/check.v1"
)

type podSuite struct{}

var _ = Suite(&podSuite{})

func (s *podSuite) TestPod20Read(c *C) {
	sourceJSON := `{"id": 42, "name": "test-pod", "type": "lxd", "resource_uri": "/MAAS/api/2.0/pods/42/", "zone": {"id": 1, "name": "zone1", "description": "", "resource_uri": "/MAAS/api/2.0/zones/zone1/"}, "pool": {"name": "pool1", "description": "", "resource_uri": "/MAAS/api/2.0/resourcepool/pool1/"}}`
	var source map[string]any
	err := json.Unmarshal([]byte(sourceJSON), &source)
	c.Assert(err, IsNil)
	p, err := pod_2_0(source)
	c.Assert(err, IsNil)
	c.Assert(p.ID(), Equals, 42)
	c.Assert(p.Name(), Equals, "test-pod")
	c.Assert(p.Type(), Equals, "lxd")
	c.Assert(p.Zone(), NotNil)
	c.Assert(p.Zone().Name(), Equals, "zone1")
	c.Assert(p.Pool(), NotNil)
	c.Assert(p.Pool().Name(), Equals, "pool1")
}
func (s *podSuite) TestPod20ReadMissingOptional(c *C) {
	sourceJSON := `{"id": 42, "name": "test-pod", "resource_uri": "/MAAS/api/2.0/pods/42/"}`
	var source map[string]any
	err := json.Unmarshal([]byte(sourceJSON), &source)
	c.Assert(err, IsNil)
	p, err := pod_2_0(source)
	c.Assert(err, IsNil)
	c.Assert(p.ID(), Equals, 42)
	c.Assert(p.Name(), Equals, "test-pod")
	c.Assert(p.Type(), Equals, "")
	c.Assert(p.Zone(), IsNil)
	c.Assert(p.Pool(), IsNil)
}
func (s *podSuite) TestReadPods(c *C) {
	sourceJSON := `[{"id": 1, "name": "pod1", "resource_uri": "/MAAS/api/2.0/pods/1/"},{"id": 2, "name": "pod2", "resource_uri": "/MAAS/api/2.0/pods/2/"}]`
	var source any
	err := json.Unmarshal([]byte(sourceJSON), &source)
	c.Assert(err, IsNil)
	pods, err := readPods(twoDotOh, source)
	c.Assert(err, IsNil)
	c.Assert(pods, HasLen, 2)
	c.Assert(pods[0].ID(), Equals, 1)
	c.Assert(pods[0].Name(), Equals, "pod1")
	c.Assert(pods[1].ID(), Equals, 2)
	c.Assert(pods[1].Name(), Equals, "pod2")
}
