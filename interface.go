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

	// Zones lists all the zones known to the MAAS controller.
	Zones() ([]Zone, error)

	// Machines returns a list of machines that match the params.
	Machines(MachinesArgs) ([]Machine, error)
}

// Zone represents a physical zone that a Machine is in. The meaning of a
// physical zone is up to you: it could identify e.g. a server rack, a network,
// or a data centre. Users can then allocate nodes from specific physical zones,
// to suit their redundancy or performance requirements.
type Zone interface {
	Name() string
	Description() string
}

// Machine represents a physical machine.
type Machine interface {
	SystemId() string
	Hostname() string
	FQDN() string
	IPAddresses() []string
	Memory() int
	CpuCount() int
	PowerState() string
	Zone() Zone
	OperatingSystem() string
	DistroSeries() string
	Architecture() string
	Status() string
}

type MachinesArgs struct {
	SystemIds []string
}
