// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/json"
	"net/url"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/schema"
	"github.com/juju/utils/set"
	"github.com/juju/version"
)

var (
	logger = loggo.GetLogger("maas")

	// The supported versions should be ordered from most desirable version to
	// least as they will be tried in order.
	supportedAPIVersions = []string{"2.0"}
)

// ControllerArgs is an argument struct for passing the required parameters
// to the NewController method.
type ControllerArgs struct {
	BaseUrl string
	APIKey  string
}

// NewController creates an authenticated client to the MAAS API, and checks
// the capabilities of the server.
func NewController(args ControllerArgs) (Controller, error) {
	// For now we don't need to test multiple versions. It is expected that at
	// some time in the future, we will try the most up to date version and then
	// work our way backwards.
	var outerErr error
	for _, apiVersion := range supportedAPIVersions {
		major, minor, err := version.ParseMajorMinor(apiVersion)
		// We should not get an error here. See the test.
		if err != nil {
			return nil, errors.Errorf("bad version defined in supported versions: %q", apiVersion)
		}
		client, err := NewAuthenticatedClient(args.BaseUrl, args.APIKey, apiVersion)
		if err != nil {
			outerErr = err
			continue
		}
		controllerVersion := version.Number{
			Major: major,
			Minor: minor,
		}
		// The controllerVersion returned from the function will include any patch version.
		capabilities, controllerVersion, err := readAPIVersion(client, controllerVersion)
		if err != nil {
			logger.Debugf("read version failed: %v", err)
			outerErr = err
			continue
		}
		return &controller{
			client:       client,
			apiVersion:   controllerVersion,
			capabilities: capabilities,
		}, nil
	}

	return nil, errors.Wrap(outerErr, errors.New("unable to create authenticated client"))
}

type controller struct {
	client       *Client
	apiVersion   version.Number
	capabilities set.Strings
}

// Capabilities implements Controller.
func (c *controller) Capabilities() set.Strings {
	return c.capabilities
}

func readAPIVersion(client *Client, apiVersion version.Number) (set.Strings, version.Number, error) {
	// So want to fix this... it is kinda horrible, will wrap it later.
	bytes, err := client.Get(&url.URL{Path: "version"}, "", nil) // perhaps "version/"
	if err != nil {
		return nil, apiVersion, errors.Trace(err)
	}

	var parsed interface{}
	err = json.Unmarshal(bytes, &parsed)
	if err != nil {
		return nil, apiVersion, errors.Trace(err)
	}

	// As we care about other fields, add them.
	fields := schema.Fields{
		"capabilities": schema.List(schema.String()),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(parsed, nil)
	if err != nil {
		return nil, apiVersion, errors.Trace(err)
	}
	// For now, we don't append any subversion, but as it becomes used, we
	// should parse and check.

	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.
	capabilities := set.NewStrings()
	capabilityValues := valid["capabilities"].([]interface{})
	for _, value := range capabilityValues {
		capabilities.Add(value.(string))
	}

	return capabilities, apiVersion, nil
}
