package main

import (
	"fmt"
	"net/url"
	"launchpad.net/gomaasapi"
)

var apiKey string
var apiURL string

func init() {
	fmt.Print("Enter API key: ")
	fmt.Scanf("%s", &apiKey)
	fmt.Print("Enter API URL: ")
	fmt.Scanf("%s", &apiURL)
}

func main() {
	authClient, err := gomaasapi.NewAuthenticatedClient(apiKey)
	if err != nil {
		panic(err)
	}

	server, err := gomaasapi.NewServer(apiURL, *authClient)

        nodeListing := server.SubObject("/nodes/")

        // List nodes.
	fmt.Println("Fetching list of nodes...")
	listNodeObjects, _ := nodeListing.CallGet("list", url.Values{})
        listNodes, err := listNodeObjects.GetArray()
	fmt.Printf("Got list of %v nodes\n", len(listNodes))
        for index, nodeObj := range(listNodes) {
            node, _ := nodeObj.GetMAASObject()
            hostname, _ := node.GetField("hostname")
	    fmt.Printf("Node #%d is named '%v' (%v)\n", index, hostname, node.URL())
        }

        // Create a node.
	fmt.Println("Creating a new node...")
        params := url.Values{"architecture": {"i386/generic"}, "mac_addresses": {"AA:BB:CC:DD:EE:FF"}}
	newNodeObj, _ := nodeListing.CallPost("new", params)
        newNode, _ := newNodeObj.GetMAASObject()
        newNodeName, _ := newNode.GetField("hostname")
	fmt.Printf("New node created: %s (%s)\n", newNodeName, newNode.URL())

        // Update the new node.
	fmt.Println("Updating the new node...")
        updateParams := url.Values{"hostname": {"mynewname"}}
	newNodeObj2, err := newNode.Update(updateParams)
        if err != nil {
            panic(err)
        }
        newNode2, _ := newNodeObj2.GetMAASObject()
        newNodeName2, _ := newNode2.GetField("hostname")
	fmt.Printf("New node updated, now named: %s\n", newNodeName2)

        // Delete the new node.
	fmt.Println("Deleting the new node...")
	errDelete := newNode.Delete()
	if errDelete != nil {
            panic(errDelete)
        }

	fmt.Println("All done.")
}
