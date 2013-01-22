// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

type Node struct {
	server       *Server `json:"-"`
	systemId     string  `json:"system_id"`
	status       int     `json:"status"`
	netboot      string  `json:"netboot"`
	hostname     string  `json:"hostname"`
	architecture string  `json:"architecture"`
	resourceUri  string  `json:"resource_uri"`
}
