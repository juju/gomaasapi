// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi_test

import (
	"github.com/juju/gomaasapi"
	gc "gopkg.in/check.v1"
)

type urlParamsSuite struct {
}

var _ = gc.Suite(&urlParamsSuite{})

func (*urlParamsSuite) TestNewParamsNonNilValues(c *gc.C) {
	params := gomaasapi.NewURLParams()
	c.Assert(params.Values, gc.NotNil)
}

func (*urlParamsSuite) TestNewMaybeAddEmpty(c *gc.C) {
	params := gomaasapi.NewURLParams()
	params.MaybeAdd("foo", "")
	c.Assert(params.Values.Encode(), gc.Equals, "")
}

func (*urlParamsSuite) TestNewMaybeAddWithValue(c *gc.C) {
	params := gomaasapi.NewURLParams()
	params.MaybeAdd("foo", "bar")
	c.Assert(params.Values.Encode(), gc.Equals, "foo=bar")
}

func (*urlParamsSuite) TestNewMaybeAddManyNil(c *gc.C) {
	params := gomaasapi.NewURLParams()
	params.MaybeAddMany("foo", nil)
	c.Assert(params.Values.Encode(), gc.Equals, "")
}

func (*urlParamsSuite) TestNewMaybeAddManyValues(c *gc.C) {
	params := gomaasapi.NewURLParams()
	params.MaybeAddMany("foo", []string{"two", "", "values"})
	c.Assert(params.Values.Encode(), gc.Equals, "foo=two&foo=values")
}
