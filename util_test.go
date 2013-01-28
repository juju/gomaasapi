// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	. "launchpad.net/gocheck"
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

func (suite *GomaasapiTestSuite) TestAppendSlashAppendsSlashIfMissing(c *C) {
	c.Check(AppendSlash("test"), Equals, "test/")
}

func (suite *GomaasapiTestSuite) TestAppendSlashDoesNotAppendsIfPresent(c *C) {
	c.Check(AppendSlash("test/"), Equals, "test/")
}

func (suite *GomaasapiTestSuite) TestAppendSlashReturnsSlashIfEmpty(c *C) {
	c.Check(AppendSlash(""), Equals, "/")
}
