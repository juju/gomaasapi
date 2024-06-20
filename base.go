// Copyright 2023 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/juju/version/v2"
)

var VersionDeploySupportsBases = version.Number{Major: 3, Minor: 3, Patch: 5}

type OSType string

var (
	Custom OSType = "custom"
	Ubuntu OSType = "ubuntu"
	Centos OSType = "centos"
)

type Base struct {
	OS      OSType
	Version string
}

func (b Base) String() string {
	if b.OS != "" && b.Version != "" {
		return fmt.Sprintf("%s/%s", b.OS, b.Version)
	}
	if b.Version != "" {
		return b.Version
	}
	return ""
}

func (b Base) toSeries() (string, error) {
	if b.OS == "" {
		return b.Version, nil
	}
	switch b.OS {
	case Custom:
		return b.String(), nil
	case Ubuntu:
		if series, ok := ubuntuMap[b.Version]; ok {
			return series, nil
		}
		return "", errors.NotValidf("base %s cannot be converted into a series", b)
	case Centos:
		if series, ok := centosMap[b.Version]; ok {
			return series, nil
		}
		return "", errors.NotValidf("base %s cannot be converted into a series", b)
	default:
		return "", errors.NotValidf("base %s OS", b)
	}

}

const (
	centos7 = "centos7"
	centos8 = "centos8"
	centos9 = "centos9"
)

var ubuntuMap = map[string]string{
	"20.04": "ubuntu/focal",
	"20.10": "ubuntu/groovy",
	"21.04": "ubuntu/hirsute",
	"21.10": "ubuntu/impish",
	"22.04": "ubuntu/jammy",
	"22.10": "ubuntu/kinetic",
	"23.04": "ubuntu/lunar",
}

var centosMap = map[string]string{
	"7": "centos/centos7",
	"8": "centos/centos8",
	"9": "centos/centos9",
}
