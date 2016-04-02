// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/json"
	"net/http"
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
//
// If the APIKey is not valid, a NotValid error is returned.
func NewController(args ControllerArgs) (Controller, error) {
	// For now we don't need to test multiple versions. It is expected that at
	// some time in the future, we will try the most up to date version and then
	// work our way backwards.
	for _, apiVersion := range supportedAPIVersions {
		major, minor, err := version.ParseMajorMinor(apiVersion)
		// We should not get an error here. See the test.
		if err != nil {
			return nil, errors.Errorf("bad version defined in supported versions: %q", apiVersion)
		}
		client, err := NewAuthenticatedClient(args.BaseURL, args.APIKey, apiVersion)
		if err != nil {
			// If the credentials aren't valid, return now.
			if errors.IsNotValid(err) {
				return nil, errors.Trace(err)
			}
			// Any other error attempting to create the authenticated client
			// is an unexpeded error and return now.
			return nil, NewUnexpectedError(err)
		}
		controllerVersion := version.Number{
			Major: major,
			Minor: minor,
		}
		controller := &controller{client: client}
		// The controllerVersion returned from the function will include any patch version.
		controller.capabilities, controller.apiVersion, err = controller.readAPIVersion(controllerVersion)
		if err != nil {
			logger.Debugf("read version failed: %#v", err)
			continue
		}
		return controller, nil
	}

	return nil, NewUnsupportedVersionError("controller at %s does not support any of %s", args.BaseURL, supportedAPIVersions)
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

// BootResources implements Controller.
func (c *controller) BootResources() ([]BootResource, error) {
	source, err := c.get("boot-resources")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	resources, err := readBootResources(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []BootResource
	for _, r := range resources {
		result = append(result, r)
	}
	return result, nil
}

// Fabrics implements Controller.
func (c *controller) Fabrics() ([]Fabric, error) {
	source, err := c.get("fabrics")
	if err != nil {
		return nil, NewUnexpectedError(err)
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

// Spaces implements Controller.
func (c *controller) Spaces() ([]Space, error) {
	source, err := c.get("spaces")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	spaces, err := readSpaces(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Space
	for _, space := range spaces {
		result = append(result, space)
	}
	return result, nil
}

// Zones implements Controller.
func (c *controller) Zones() ([]Zone, error) {
	source, err := c.get("zones")
	if err != nil {
		return nil, NewUnexpectedError(err)
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

// MachinesArgs is a argument struct for selecting Machines.
// Only machines that match the specified criteria are returned.
type MachinesArgs struct {
	Hostnames    []string
	MACAddresses []string
	SystemIds    []string
	Domain       string
	Zone         string
	AgentName    string
}

// Machines implements Controller.
func (c *controller) Machines(args MachinesArgs) ([]Machine, error) {
	params := NewURLParams()
	params.MaybeAddMany("hostname", args.Hostnames)
	params.MaybeAddMany("mac_address", args.MACAddresses)
	params.MaybeAddMany("id", args.SystemIds)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAdd("zone", args.Zone)
	params.MaybeAdd("agent_name", args.AgentName)
	source, err := c.getQuery("machines", params.Values)
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	machines, err := readMachines(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Machine
	for _, m := range machines {
		m.controller = c
		result = append(result, m)
	}
	return result, nil
}

// AllocateMachineArgs is an argument struct for passing args into Machine.Allocate.
type AllocateMachineArgs struct {
	Hostname     string
	Architecture string
	MinCPUCount  int
	// MinMemory represented in MB.
	MinMemory int
	Tags      []string
	NotTags   []string
	// Networks - list of networks (defined in MAAS) to which the machine must be
	// attached. A network can be identified by the name assigned to it in MAAS;
	// or by an ip: prefix followed by any IP address that falls within the
	// network; or a vlan: prefix followed by a numeric VLAN tag, e.g. vlan:23
	// for VLAN number 23. Valid VLAN tags must be in the range of 1 to 4094
	// inclusive.
	Networks    []string
	NotNetworks []string
	Zone        string
	NotInZone   []string
	AgentName   string
	Comment     string
	DryRun      bool
}

// AllocateMachine implements Controller.
//
// Returns an error that satisfies IsNoMatchError if the requested
// constraints cannot be met.
func (c *controller) AllocateMachine(args AllocateMachineArgs) (Machine, error) {
	params := NewURLParams()
	params.MaybeAdd("name", args.Hostname)
	params.MaybeAdd("arch", args.Architecture)
	params.MaybeAddInt("cpu_count", args.MinCPUCount)
	params.MaybeAddInt("mem", args.MinMemory)
	params.MaybeAddMany("tags", args.Tags)
	params.MaybeAddMany("not_tags", args.NotTags)
	params.MaybeAddMany("networks", args.Networks)
	params.MaybeAddMany("not_networks", args.NotNetworks)
	params.MaybeAdd("zone", args.Zone)
	params.MaybeAddMany("not_in_zone", args.NotInZone)
	params.MaybeAdd("agent_name", args.AgentName)
	params.MaybeAdd("comment", args.Comment)
	params.MaybeAddBool("dry_run", args.DryRun)
	result, err := c.post("machines", "allocate", params.Values)
	if err != nil {
		// A 409 Status code is "No Matching Machines"
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusConflict {
				return nil, errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			}
		}
		// Translate http errors.
		return nil, NewUnexpectedError(err)
	}

	machine, err := readMachine(c.apiVersion, result)
	if err != nil {
		return nil, errors.Trace(err)
	}
	machine.controller = c
	return machine, nil
}

// ReleaseMachinesArgs is an argument struct for passing the machine system IDs
// and an optional comment into the ReleaseMachines method.
type ReleaseMachinesArgs struct {
	SystemIDs []string
	Comment   string
}

// ReleaseMachines implements Controller.
//
// Release multiple machines at once. Returns
//  - BadRequestError if any of the machines cannot be found
//  - PermissionError if the user does not have permission to release any of the machines
//  - CannotCompleteError if any of the machines could not be released due to their current state
func (c *controller) ReleaseMachines(args ReleaseMachinesArgs) error {
	params := NewURLParams()
	params.MaybeAddMany("machines", args.SystemIDs)
	params.MaybeAdd("comment", args.Comment)
	_, err := c.post("machines", "release", params.Values)
	if err != nil {
		// A 409 Status code is "No Matching Machines"
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusBadRequest {
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			}
			if svrErr.StatusCode == http.StatusForbidden {
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			}
			if svrErr.StatusCode == http.StatusConflict {
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		// Translate http errors.
		return NewUnexpectedError(err)
	}

	return nil
}

func (c *controller) post(path, op string, params url.Values) (interface{}, error) {
	path = EnsureTrailingSlash(path)
	requestID := nextrequestID()
	logger.Tracef("request %x: POST %s%s?op=%s, params=%s", requestID, c.client.APIURL, path, op, params.Encode())
	bytes, err := c.client.Post(&url.URL{Path: path}, op, params, nil)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
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

func (c *controller) getQuery(path string, params url.Values) (interface{}, error) {
	return c._get(path, params)
}

func (c *controller) get(path string) (interface{}, error) {
	return c._get(path, nil)
}

func (c *controller) _get(path string, params url.Values) (interface{}, error) {
	path = EnsureTrailingSlash(path)
	requestID := nextrequestID()
	if logger.IsTraceEnabled() {
		var query string
		if params != nil {
			query = "?" + params.Encode()
		}
		logger.Tracef("request %x: GET %s%s%s", requestID, c.client.APIURL, path, query)
	}
	bytes, err := c.client.Get(&url.URL{Path: path}, "", params)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
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
		return nil, apiVersion, WrapWithDeserializationError(err, "version response")
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
