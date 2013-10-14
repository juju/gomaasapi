// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

const (
	// NodeStatus* values represent the vocabulary of a Node‘s possible statuses.

	// The node has been created and has a system ID assigned to it.
	NodeStatusDeclared = "0"

	//Testing and other commissioning steps are taking place.
	NodeStatusCommissioning = "1"

	// Smoke or burn-in testing has a found a problem.
	NodeStatusFailedTests = "2"

	// The node can’t be contacted.
	NodeStatusMissing = "3"

	// The node is in the general pool ready to be deployed.
	NodeStatusReady = "4"

	// The node is ready for named deployment.
	NodeStatusReserved = "5"

	// The node is powering a service from a charm or is ready for use with a fresh Ubuntu install.
	NodeStatusAllocated = "6"

	// The node has been removed from service manually until an admin overrides the retirement.
	NodeStatusRetired = "7"
)
