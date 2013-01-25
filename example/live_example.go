package main

import (
	"fmt"
	"launchpad.net/gomaasapi"
)

var apiKey string
var apiURL string

func init() {
	fmt.Print("Enter apiKey: ")
	fmt.Scanf("%s", &apiKey)
	fmt.Print("Enter apiURL: ")
	fmt.Scanf("%s", &apiURL)
}

func main() {
	authClient, err := gomaasapi.NewAuthenticatedClient(apiKey)
	if err != nil {
		panic(err)
	}

	server := gomaasapi.Server{apiURL, authClient}
	fmt.Println("Fetching list of nodes...")
	listNodes, _ := server.ListNodes()
	fmt.Printf("Got list of nodes: %s\n", listNodes)
}
