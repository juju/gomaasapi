// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"fmt"
	"net/url"
)

func NewServer(URL string, client Client) (MAASObject, error) {
	parsed, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	resourceURI := parsed.Path
	input := map[string]JSONObject{resource_uri: jsonString(resourceURI)}
	return jsonMAASObject{jsonMap: jsonMap(input), client: client, baseURL: baseURL}, nil
}
