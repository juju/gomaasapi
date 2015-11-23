// Copyright 2015 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

func getSpacesEndpoint(version string) string {
	return fmt.Sprintf("/api/%s/spaces/", version)
}

// Space is the MAAS API space representation
type Space struct {
	Name        string   `json:"name"`
	Subnets     []Subnet `json:"subnets"`
	ResourceURI string   `json:"resource_uri"`
	ID          uint     `json:"id"`
}

// spacesHandler handles requests for '/api/<version>/spaces/'.
func spacesHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	var err error
	spacesURLRE := regexp.MustCompile(`/spaces/(.+?)/`)
	spacesURLMatch := spacesURLRE.FindStringSubmatch(r.URL.Path)
	spacesURL := getSpacesEndpoint(server.version)

	var ID uint
	var gotID bool
	if spacesURLMatch != nil {
		ID, err = NameOrIDToID(spacesURLMatch[1], server.spaceNameToID, 1, uint(len(server.spaces)))

		if err != nil {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		gotID = true
	}

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if len(server.spaces) == 0 {
			// Until a space is registered, behave as if the endpoint
			// does not exist. This way we can simulate older MAAS
			// servers that do not support spaces.
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		if r.URL.Path == spacesURL {
			var spaces []Space
			for i := uint(1); i < server.nextSpace; i++ {
				s, ok := server.spaces[i]
				if ok {
					spaces = append(spaces, s)
				}
			}
			err = json.NewEncoder(w).Encode(spaces)
		} else if gotID == false {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			err = json.NewEncoder(w).Encode(server.spaces[ID])
		}
		checkError(err)
	case "POST":
		//server.NewSpace(r.Body)
	case "PUT":
		//server.UpdateSpace(r.Body)
	case "DELETE":
		delete(server.spaces, ID)
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
