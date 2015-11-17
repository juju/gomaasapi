// Copyright 2015 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

func getSubnetsEndpoint(version string) string {
	return fmt.Sprintf("/api/%s/subnets/", version)
}

// CreateSubnet is used to receive new subnets via the MAAS API
type CreateSubnet struct {
	DNSServers []string `json:"dns_servers"`
	Name       string   `json:"name"`
	Space      string   `json:"space"`
	GatewayIP  string   `json:"gateway_ip"`
	CIDR       string   `json:"cidr"`

	// VLAN this subnet belongs to. Defaults to the default VLAN
	// for the provided fabric or defaults to the default VLAN
	// in the default fabric.
	VLAN *uint `json:"vlan"`

	// Fabric for the subnet. Defaults to the fabric the provided
	// VLAN belongs to or defaults to the default fabric.
	Fabric *uint `json:"fabric"`

	// VID of the VLAN this subnet belongs to. Only used when vlan
	// is not provided. Picks the VLAN with this VID in the provided
	// fabric or the default fabric if one is not given.
	VID *uint `json:"vid"`

	// This is used for updates (PUT) and is ignored by create (POST)
	ID int `json:"id"`
}

// Subnet is the MAAS API subnet representation
type Subnet struct {
	DNSServers []string `json:"dns_servers"`
	Name       string   `json:"name"`
	Space      string   `json:"string"`
	VLAN       VLAN     `json:"vlan"`
	GatewayIP  string   `json:"gateway_ip"`
	CIDR       string   `json:"cidr"`

	ResourceURI string `json:"resource_uri"`
	ID          int    `json:"id"`
}

// subnetsHandler handles requests for '/api/<version>/subnets/'.
func subnetsHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	var err error
	/*values, err := url.ParseQuery(r.URL.RawQuery)
	checkError(err)
	op := values.Get("op")*/
	subnetsURLRE := regexp.MustCompile(`/subnets/(\d+)/`)
	subnetsURLMatch := subnetsURLRE.FindStringSubmatch(r.URL.Path)
	subnetsURL := getSubnetsEndpoint(server.version)

	var ID int
	var gotID bool
	if subnetsURLMatch != nil {
		ID, err = strconv.Atoi(subnetsURLMatch[1])
		checkError(err)

		if len(server.subnets) < ID-1 || ID == 0 {
			// IDs start at 1...
			w.WriteHeader(http.StatusBadRequest)
		}

		gotID = true
	}

	switch r.Method {
	case "GET":
		if len(server.subnets) == 0 {
			// Until a subnet is registered, behave as if the endpoint
			// does not exist. This way we can simulate older MAAS
			// servers that do not support subnets.
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		if r.URL.Path == subnetsURL {
			var subnets []Subnet
			for i := 1; i < server.nextSubnet; i++ {
				s, ok := server.subnets[i]
				if ok {
					subnets = append(subnets, s)
				}
			}
			err = json.NewEncoder(w).Encode(subnets)
		} else if gotID == false {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			err = json.NewEncoder(w).Encode(server.subnets[ID])
		}
		checkError(err)
	case "POST":
		server.NewSubnet(r.Body)
	case "PUT":
		server.UpdateSubnet(r.Body)
	case "DELETE":
		delete(server.subnets, ID)
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func decodePostedSubnet(subnetJSON io.Reader) CreateSubnet {
	var postedSubnet CreateSubnet
	decoder := json.NewDecoder(subnetJSON)
	err := decoder.Decode(&postedSubnet)
	checkError(err)
	return postedSubnet
}

// UpdateSubnet creates a subnet in the test server
func (server *TestServer) UpdateSubnet(subnetJSON io.Reader) Subnet {
	postedSubnet := decodePostedSubnet(subnetJSON)
	updatedSubnet := subnetFromCreateSubnet(postedSubnet)
	server.subnets[updatedSubnet.ID] = updatedSubnet
	return updatedSubnet
}

// NewSubnet creates a subnet in the test server
func (server *TestServer) NewSubnet(subnetJSON io.Reader) Subnet {
	postedSubnet := decodePostedSubnet(subnetJSON)
	newSubnet := subnetFromCreateSubnet(postedSubnet)
	newSubnet.ID = server.nextSubnet
	server.subnets[server.nextSubnet] = newSubnet
	server.nextSubnet++
	return newSubnet
}

// subnetFromCreateSubnet creates a subnet in the test server
func subnetFromCreateSubnet(postedSubnet CreateSubnet) Subnet {
	var newSubnet Subnet
	newSubnet.DNSServers = postedSubnet.DNSServers
	newSubnet.Name = postedSubnet.Name
	newSubnet.Space = postedSubnet.Space
	//TODO: newSubnet.VLAN = server.postedSubnetVLAN
	newSubnet.GatewayIP = postedSubnet.GatewayIP
	newSubnet.CIDR = postedSubnet.CIDR
	newSubnet.ID = postedSubnet.ID
	return newSubnet
}

func subnetsHandlerDelete(server *TestServer, w http.ResponseWriter, r *http.Request) {
}
