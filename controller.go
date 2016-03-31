// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/json"
	"net/url"
	"sync/atomic"

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

	// Each of the api versions that change the request or response structure
	// for any given call should have a value defined for easy definition of
	// the deserialization functions.
	twoDotOh = version.Number{Major: 2, Minor: 0}

	// Current request number. Informational only for logging.
	requestNumber int64
)

// ControllerArgs is an argument struct for passing the required parameters
// to the NewController method.
type ControllerArgs struct {
	BaseURL string
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
		client, err := NewAuthenticatedClient(args.BaseURL, args.APIKey, apiVersion)
		if err != nil {
			outerErr = err
			continue
		}
		controllerVersion := version.Number{
			Major: major,
			Minor: minor,
		}
		controller := &controller{client: client}
		// The controllerVersion returned from the function will include any patch version.
		controller.capabilities, controller.apiVersion, err = controller.readAPIVersion(controllerVersion)
		if err != nil {
			logger.Debugf("read version failed: %v", err)
			outerErr = err
			continue
		}
		return controller, nil
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

// Zones implements Controller.
func (c *controller) Fabrics() ([]Fabric, error) {
	source, err := c.get("fabrics")
	if err != nil {
		return nil, errors.Trace(err)
	}
	fabrics, err := readFabrics(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Fabric
	for _, f := range fabrics {
		result = append(result, f)
	}
	return result, nil
}

// Zones implements Controller.
func (c *controller) Zones() ([]Zone, error) {
	source, err := c.get("zones")
	if err != nil {
		return nil, errors.Trace(err)
	}
	zones, err := readZones(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Zone
	for _, z := range zones {
		result = append(result, z)
	}
	return result, nil
}

// Machines implements Controller.
func (c *controller) Machines(params MachinesArgs) ([]Machine, error) {
	// ignore params for now
	source, err := c.get("machines")
	if err != nil {
		return nil, errors.Trace(err)
	}
	machines, err := readMachines(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Machine
	for _, m := range machines {
		result = append(result, m)
	}
	return result, nil
}

func (c *controller) get(path string) (interface{}, error) {
	path = EnsureTrailingSlash(path)
	requestID := nextrequestID()
	logger.Tracef("request %x: GET %s%s", requestID, c.client.APIURL, path)
	bytes, err := c.client.Get(&url.URL{Path: path}, "", nil)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		return nil, errors.Trace(err)
	}
	logger.Tracef("response %x: %s", requestID, string(bytes))

	var parsed interface{}
	err = json.Unmarshal(bytes, &parsed)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return parsed, nil
}

func nextrequestID() int64 {
	return atomic.AddInt64(&requestNumber, 1)
}

func (c *controller) readAPIVersion(apiVersion version.Number) (set.Strings, version.Number, error) {
	parsed, err := c.get("version")
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
