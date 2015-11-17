// Copyright 2015 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"
	"io"
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
	switch r.Method {
	case "GET":
		vlansHandlerGet(server, w, r)
	case "POST":
		vlansHandlerPost(server, w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func vlansHandlerGet(server *TestServer, w http.ResponseWriter, r *http.Request) {
	if len(server.vlans) == 0 {
		// Until a subnet is registered, behave as if the endpoint
		// does not exist. This way we can simulate older MAAS
		// servers that do not support vlans.
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	err := json.NewEncoder(w).Encode(server.vlans)
	checkError(err)
}

func vlansHandlerPost(server *TestServer, w http.ResponseWriter, r *http.Request) {
	server.NewSubnet(r.Body)
}

// NewVLAN creates a new VLAN in the test server
func (server *TestServer) NewVLAN(postedJSON io.Reader) VLAN {
	var v VLAN
	decoder := json.NewDecoder(postedJSON)
	err := decoder.Decode(&v)
	checkError(err)
	return server.NewVLANFromVLAN(v)
}

// NewVLANFromVLAN creates a new VLAN in the test server
func (server *TestServer) NewVLANFromVLAN(vlan VLAN) VLAN {
	server.vlans[server.nextVLAN] = vlan
	server.nextVLAN++
	return vlan
}

func postedSubnetVLAN(postedSubnet *CreateSubnet) VLAN {
	var vlan VLAN
	// VLAN this subnet belongs to. Defaults to the default VLAN
	// for the provided fabric or defaults to the default VLAN
	// in the default fabric.
	if postedSubnet.VLAN == nil {
		if postedSubnet.Fabric == nil {
			//TODO: default Fabric...
			if postedSubnet.VID == nil {
				panic("Need VLAN, Fabric or VID")
			}
		} else {
			//TODO: VLAN of postedSubnet.Fabric
		}
		// VID of the VLAN this subnet belongs to. Only used when vlan
		// is not provided. Picks the VLAN with this VID in the provided
		// fabric or the default fabric if one is not given.
		//postedSubnet.VID
	}

	// Fabric for the subnet. Defaults to the fabric the provided
	// VLAN belongs to or defaults to the default fabric.
	if postedSubnet.Fabric == nil {
		if postedSubnet.VLAN == nil {
			//Default fabric
		} else {
			//VLAN provided fabric
		}
	}

	//if postedSubnet.VID == 0

	return vlan
}
