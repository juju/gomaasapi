package gomaasapi

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type GomaasapiTestSuite struct {
}

var _ = Suite(&GomaasapiTestSuite{})
