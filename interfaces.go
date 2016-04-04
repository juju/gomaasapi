// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import "github.com/juju/utils/set"

const (
	// Capability constants.
	NetworksManagement      = "networks-management"
	StaticIPAddresses       = "static-ipaddresses"
	IPv6DeploymentUbuntu    = "ipv6-deployment-ubuntu"
	DevicesManagement       = "devices-management"
	StorageDeploymentUbuntu = "storage-deployment-ubuntu"
	NetworkDeploymentUbuntu = "network-deployment-ubuntu"
)

// Controller represents an API connection to a MAAS Controller. Since the API
// is restful, there is no long held connection to the API server, but instead
// HTTP calls are made and JSON response structures parsed.
type Controller interface {

	// Capabilities returns a set of capabilities as defined by the string
	// constants.
	Capabilities() set.Strings

	BootResources() ([]BootResource, error)

	// Fabrics returns the list of Fabrics defined in the MAAS controller.
	Fabrics() ([]Fabric, error)

	// Spaces returns the list of Spaces defined in the MAAS controller.
	Spaces() ([]Space, error)

	// Zones lists all the zones known to the MAAS controller.
	Zones() ([]Zone, error)

	// Machines returns a list of machines that match the params.
	Machines(MachinesArgs) ([]Machine, error)

	AllocateMachine(AllocateMachineArgs) (Machine, error)
	ReleaseMachines(ReleaseMachinesArgs) error

	// Devices returns a list of devices that match the params.
	Devices(DevicesArgs) ([]Device, error)
	CreateDevice(CreateDeviceArgs) (Device, error)
}

// Fabric represents a set of interconnected VLANs that are capable of mutual
// communication. A fabric can be thought of as a logical grouping in which
// VLANs can be considered unique.
//
// For example, a distributed network may have a fabric in London containing
// VLAN 100, while a separate fabric in San Francisco may contain a VLAN 100,
// whose attached subnets are completely different and unrelated.
type Fabric interface {
	ID() int
	Name() string
	ClassType() string

	VLANs() []VLAN
}

// VLAN represents an instance of a Virtual LAN. VLANs are a common way to
// create logically separate networks using the same physical infrastructure.
//
// Managed switches can assign VLANs to each port in either a “tagged” or an
// “untagged” manner. A VLAN is said to be “untagged” on a particular port when
// it is the default VLAN for that port, and requires no special configuration
// in order to access.
//
// “Tagged” VLANs (traditionally used by network administrators in order to
// aggregate multiple networks over inter-switch “trunk” lines) can also be used
// with nodes in MAAS. That is, if a switch port is configured such that
// “tagged” VLAN frames can be sent and received by a MAAS node, that MAAS node
// can be configured to automatically bring up VLAN interfaces, so that the
// deployed node can make use of them.
//
// A “Default VLAN” is created for every Fabric, to which every new VLAN-aware
// object in the fabric will be associated to by default (unless otherwise
// specified).
type VLAN interface {
	ID() int
	Name() string
	Fabric() string

	// VID is the VLAN ID. eth0.10 -> VID = 10.
	VID() int
	// MTU (maximum transmission unit) is the largest size packet or frame,
	// specified in octets (eight-bit bytes), that can be sent.
	MTU() int
	DHCP() bool

	PrimaryRack() string
	SecondaryRack() string
}

// Zone represents a physical zone that a Machine is in. The meaning of a
// physical zone is up to you: it could identify e.g. a server rack, a network,
// or a data centre. Users can then allocate nodes from specific physical zones,
// to suit their redundancy or performance requirements.
type Zone interface {
	Name() string
	Description() string
}

// BootResource is the bomb... find something to say here.
type BootResource interface {
	ID() int
	Name() string
	Type() string
	Architecture() string
	SubArchitectures() set.Strings
	KernelFlavor() string
}

// Device represents some form of device in MAAS.
type Device interface {
	// TODO: add domain
	SystemID() string
	Hostname() string
	FQDN() string

	IPAddresses() []string

	Zone() Zone

	// Parent, Owner, MAC Addresses if needed

	Delete() error
}

// Machine represents a physical machine.
type Machine interface {
	SystemID() string
	Hostname() string
	FQDN() string

	OperatingSystem() string
	DistroSeries() string
	Architecture() string
	Memory() int
	CPUCount() int

	IPAddresses() []string
	PowerState() string

	// Consider bundling the status values into a single struct.
	// but need to check for consistent representation if exposed on other
	// entities.

	StatusName() string
	StatusMessage() string

	Zone() Zone

	Start(StartArgs) error
}

// Space is a name for a collection of Subnets.
type Space interface {
	ID() int
	Name() string
	Subnets() []Subnet
}

// Subnet refers to an IP range on a VLAN.
type Subnet interface {
	ID() int
	Name() string
	Space() string
	VLAN() VLAN

	Gateway() string
	CIDR() string
	// DNS Servers, rdns_mode
}

// A link represents a network link between an Interface and a Subnet.
type Link interface {
	ID() int
	Mode() string
	Subnet() Subnet
}
