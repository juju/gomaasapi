// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"strings"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type errorTypesSuite struct{}

var _ = gc.Suite(&errorTypesSuite{})

func (*errorTypesSuite) TestNoMatchError(c *gc.C) {
	err := NewNoMatchError("foo")
	c.Assert(err, gc.NotNil)
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (*errorTypesSuite) TestUnexpectedError(c *gc.C) {
	err := errors.New("wat")
	err = NewUnexpectedError(err)
	c.Assert(err, gc.NotNil)
	c.Assert(err, jc.Satisfies, IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: wat")
}

func (*errorTypesSuite) TestUnsupportedVersionError(c *gc.C) {
	err := NewUnsupportedVersionError("foo %d", 42)
	c.Assert(err, gc.NotNil)
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
	c.Assert(err.Error(), gc.Equals, "foo 42")
}

func (*errorTypesSuite) TestDeserializationError(c *gc.C) {
	err := NewDeserializationError("foo %d", 42)
	c.Assert(err, gc.NotNil)
	c.Assert(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, "foo 42")
}

func (*errorTypesSuite) TestWrapWithDeserializationError(c *gc.C) {
	err := errors.New("base error")
	err = WrapWithDeserializationError(err, "foo %d", 42)
	c.Assert(err, gc.NotNil)
	c.Assert(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, "foo 42: base error")
	stack := errors.ErrorStack(err)
	c.Assert(strings.Split(stack, "\n"), gc.HasLen, 2)
}
