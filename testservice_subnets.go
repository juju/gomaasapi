// Copyright 2015 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
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

	// VLAN this subnet belongs to. Currently ignored.
	// TODO: Defaults to the default VLAN
	// for the provided fabric or defaults to the default VLAN
	// in the default fabric.
	VLAN *uint `json:"vlan"`

	// Fabric for the subnet. Currently ignored.
	// TODO: Defaults to the fabric the provided
	// VLAN belongs to or defaults to the default fabric.
	Fabric *uint `json:"fabric"`

	// VID of the VLAN this subnet belongs to. Currently ignored.
	// TODO: Only used when vlan
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

	ResourceURI      string `json:"resource_uri"`
	ID               int    `json:"id"`
	InUseIPAddresses []IP   `json:"-"`
}

// subnetsHandler handles requests for '/api/<version>/subnets/'.
func subnetsHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	var err error
	values, err := url.ParseQuery(r.URL.RawQuery)
	checkError(err)
	op := values.Get("op")
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
			return
		}

		gotID = true
	}

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/vnd.api+json")
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
			switch op {
			case "unreserved_ip_ranges":
				err = json.NewEncoder(w).Encode(
					server.subnetUnreservedIPRanges(server.subnets[ID]))
			case "reserved_ip_ranges":
				err = json.NewEncoder(w).Encode(
					server.subnetReservedIPRanges(server.subnets[ID]))
			default:
				err = json.NewEncoder(w).Encode(server.subnets[ID])
			}
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

// IP is an enhanced net.IP
type IP struct {
	netIP net.IP
}

// IPFromNetIP creates a IP from a net.IP.
func IPFromNetIP(netIP net.IP) IP {
	var ip IP
	ip.netIP = netIP
	return ip
}

// To4 converts the IPv4 address ip to a 4-byte representation. If ip is not
// an IPv4 address, To4 returns nil.
func (ip IP) To4() net.IP {
	return ip.netIP.To4()
}

// To16 converts the IP address ip to a 16-byte representation. If ip is not
// an IP address (it is the wrong length), To16 returns nil.
func (ip IP) To16() net.IP {
	return ip.netIP.To16()
}

func (ip IP) String() string {
	return ip.netIP.String()
}

// UInt64 returns a uint64 holding the IP address
func (ip IP) UInt64() uint64 {
	var bb *bytes.Reader
	if ip.To4() != nil {
		var v uint32
		bb = bytes.NewReader(ip.To4())
		err := binary.Read(bb, binary.BigEndian, &v)
		checkError(err)
		return uint64(v)
	}

	var v uint64
	bb = bytes.NewReader(ip.To16())
	err := binary.Read(bb, binary.BigEndian, &v)
	checkError(err)
	return v
}

// SetUInt64 sets the IP value to v
func (ip *IP) SetUInt64(v uint64) {
	bb := new(bytes.Buffer)

	if len(ip.netIP) == 0 {
		// If we don't have allocated storage make an educated guess
		// at if the address we received is an IPv4 or IPv6 address.
		if v == (v & 0x00000000ffffFFFF) {
			// Guessing IPv4
			ip.netIP = net.ParseIP("0.0.0.0")
		} else {
			ip.netIP = net.ParseIP("2001:4860:0:2001::68")
		}
	}

	var first int
	if ip.To4() != nil {
		binary.Write(bb, binary.BigEndian, uint32(v))
		first = len(ip.netIP) - 4
	} else {
		binary.Write(bb, binary.BigEndian, v)
	}
	copy(ip.netIP[first:], bb.Bytes())
}

type addressList []IP

func (a addressList) Len() int           { return len(a) }
func (a addressList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a addressList) Less(i, j int) bool { return a[i].UInt64() < a[j].UInt64() }

// AddressRange is used to generate reserved IP address range lists
type AddressRange struct {
	Start        string   `json:"start"`
	End          string   `json:"end"`
	Purpose      []string `json:"purpose,omitempty"`
	NumAddresses uint     `json:"num_addresses"`
}

func (server *TestServer) subnetUnreservedIPRanges(subnet Subnet) []AddressRange {
	// Make a sorted copy of subnet.InUseIPAddresses
	ipAddresses := make([]IP, len(subnet.InUseIPAddresses))
	copy(ipAddresses, subnet.InUseIPAddresses)
	sort.Sort(addressList(ipAddresses))

	// We need the first and last address in the subnet
	var ranges []AddressRange
	var i AddressRange
	var startIP, endIP IP

	_, ipNet, err := net.ParseCIDR(subnet.CIDR)
	checkError(err)
	startIP = IPFromNetIP(ipNet.IP)
	// Start with the lowest usable address in the range, which is 1 above
	// what net.ParseCIDR will give back.
	startIP.SetUInt64(startIP.UInt64() + 1)

	for _, endIP = range ipAddresses {
		end := endIP.UInt64()
		endIP.SetUInt64(end - 1)
		i.Start, i.End = startIP.String(), endIP.String()
		i.NumAddresses = uint(1 + endIP.UInt64() - startIP.UInt64())
		ranges = append(ranges, i)
		startIP.SetUInt64(end + 1)
	}

	ones, bits := ipNet.Mask.Size()
	set := ^((^uint64(0)) << uint(bits-ones))

	// The last usable address is one below the broadcast address, which is
	// what you get by bitwise ORing 'set' with any IP address in the subnet.
	endIP.SetUInt64((endIP.UInt64() | set) - 1)
	i.Start, i.End = startIP.String(), endIP.String()
	i.NumAddresses = uint(1 + endIP.UInt64() - startIP.UInt64())
	ranges = append(ranges, i)

	return ranges
}

func (server *TestServer) subnetReservedIPRanges(subnet Subnet) []AddressRange {
	// Make a sorted copy of subnet.InUseIPAddresses
	ipAddresses := make([]IP, len(subnet.InUseIPAddresses))
	copy(ipAddresses, subnet.InUseIPAddresses)
	sort.Sort(addressList(ipAddresses))

	var ranges []AddressRange
	var i AddressRange
	startIP := ipAddresses[0]
	var thisIP IP
	var lastIP uint64
	var startIPValid bool

	for _, thisIP = range ipAddresses {
		ip := thisIP.UInt64()
		if startIPValid == false {
			startIP.SetUInt64(ip)
			startIPValid = true
		} else if ip != lastIP && ip != lastIP+1 {
			thisIP.SetUInt64(lastIP)
			i.Start, i.End = startIP.String(), thisIP.String()
			i.NumAddresses = uint(1 + thisIP.UInt64() - startIP.UInt64())
			ranges = append(ranges, i)
			startIP.SetUInt64(ip + 1)
			startIPValid = false
		}
		lastIP = ip
	}
	if startIPValid {
		thisIP.SetUInt64(lastIP)
		i.Start, i.End = startIP.String(), thisIP.String()
		i.NumAddresses = uint(1 + thisIP.UInt64() - startIP.UInt64())
		ranges = append(ranges, i)
	}

	return ranges
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
	server.subnetNameToID[newSubnet.Name] = newSubnet.ID

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
