// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	. "gopkg.in/check.v1"
)

func (suite *GomaasapiTestSuite) TestJoinURLsAppendsPathToBaseURL(c *C) {
	c.Check(JoinURLs("http://example.com/", "foo"), Equals, "http://example.com/foo")
}

func (suite *GomaasapiTestSuite) TestJoinURLsAddsSlashIfNeeded(c *C) {
	c.Check(JoinURLs("http://example.com/foo", "bar"), Equals, "http://example.com/foo/bar")
}

func (suite *GomaasapiTestSuite) TestJoinURLsNormalizesDoubleSlash(c *C) {
	c.Check(JoinURLs("http://example.com/base/", "/szot"), Equals, "http://example.com/base/szot")
}

func (suite *GomaasapiTestSuite) TestEnsureTrailingSlashAppendsSlashIfMissing(c *C) {
	c.Check(EnsureTrailingSlash("test"), Equals, "test/")
}

func (suite *GomaasapiTestSuite) TestEnsureTrailingSlashDoesNotAppendIfPresent(c *C) {
	c.Check(EnsureTrailingSlash("test/"), Equals, "test/")
}

func (suite *GomaasapiTestSuite) TestEnsureTrailingSlashReturnsSlashIfEmpty(c *C) {
	c.Check(EnsureTrailingSlash(""), Equals, "/")
}
