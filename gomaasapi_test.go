// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type GomaasapiTestSuite struct {
}

var _ = Suite(&GomaasapiTestSuite{})
