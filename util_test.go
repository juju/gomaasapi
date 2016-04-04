// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/json"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

func (suite *GomaasapiTestSuite) TestJoinURLsAppendsPathToBaseURL(c *gc.C) {
	c.Check(JoinURLs("http://example.com/", "foo"), gc.Equals, "http://example.com/foo")
}

func (suite *GomaasapiTestSuite) TestJoinURLsAddsSlashIfNeeded(c *gc.C) {
	c.Check(JoinURLs("http://example.com/foo", "bar"), gc.Equals, "http://example.com/foo/bar")
}

func (suite *GomaasapiTestSuite) TestJoinURLsNormalizesDoubleSlash(c *gc.C) {
	c.Check(JoinURLs("http://example.com/base/", "/szot"), gc.Equals, "http://example.com/base/szot")
}

func (suite *GomaasapiTestSuite) TestEnsureTrailingSlashAppendsSlashIfMissing(c *gc.C) {
	c.Check(EnsureTrailingSlash("test"), gc.Equals, "test/")
}

func (suite *GomaasapiTestSuite) TestEnsureTrailingSlashDoesNotAppendIfPresent(c *gc.C) {
	c.Check(EnsureTrailingSlash("test/"), gc.Equals, "test/")
}

func (suite *GomaasapiTestSuite) TestEnsureTrailingSlashReturnsSlashIfEmpty(c *gc.C) {
	c.Check(EnsureTrailingSlash(""), gc.Equals, "/")
}

func parseJSON(c *gc.C, source string) interface{} {
	var parsed interface{}
	err := json.Unmarshal([]byte(source), &parsed)
	c.Assert(err, jc.ErrorIsNil)
	return parsed
}

func updateJSONMap(c *gc.C, source string, changes map[string]interface{}) string {
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(source), &parsed)
	c.Assert(err, jc.ErrorIsNil)
	for key, value := range changes {
		parsed[key] = value
	}
	bytes, err := json.Marshal(parsed)
	c.Assert(err, jc.ErrorIsNil)
	return string(bytes)
}
