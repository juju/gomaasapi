// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

var spacesResponse = `
[
    {
        "resource_uri": "/MAAS/api/2.0/spaces/0/",
        "subnets": [
            {
                "resource_uri": "/MAAS/api/2.0/subnets/34/",
                "id": 34,
                "rdns_mode": 2,
                "vlan": {
                    "resource_uri": "/MAAS/api/2.0/vlans/5001/",
                    "id": 5001,
                    "secondary_rack": null,
                    "mtu": 1500,
                    "primary_rack": null,
                    "name": "untagged",
                    "fabric": "fabric-1",
                    "dhcp_on": false,
                    "vid": 0
                },
                "dns_servers": [],
                "space": "space-0",
                "name": "192.168.122.0/24",
                "gateway_ip": null,
                "cidr": "192.168.122.0/24"
            },
            {
                "resource_uri": "/MAAS/api/2.0/subnets/1/",
                "id": 1,
                "rdns_mode": 2,
                "vlan": {
                    "resource_uri": "/MAAS/api/2.0/vlans/1/",
                    "id": 1,
                    "secondary_rack": null,
                    "mtu": 1500,
                    "primary_rack": "4y3h7n",
                    "name": "untagged",
                    "fabric": "fabric-0",
                    "dhcp_on": true,
                    "vid": 0
                },
                "dns_servers": [],
                "space": "space-0",
                "name": "192.168.100.0/24",
                "gateway_ip": "192.168.100.1",
                "cidr": "192.168.100.0/24"
            }
        ],
        "id": 0,
        "name": "space-0"
    }
]
`
