// Copyright 2015 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"fmt"
	"net/http"
)

func getVLANsEndpoint(version string) string {
	return fmt.Sprintf("/api/%s/vlans/", version)
}

// VLAN is the MAAS API VLAN representation
type VLAN struct {
	Name   string `json:"name"`
	Fabric string `json:"fabric"`
	VID    uint   `json:"vid"`

	ResourceURI string `json:"resource_uri"`
	ID          uint   `json:"id"`
}

// PostedVLAN is the MAAS API posted VLAN representation
type PostedVLAN struct {
	Name string `json:"name"`
	VID  uint   `json:"vid"`
}

func vlansHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	//TODO
}
