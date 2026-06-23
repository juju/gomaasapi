// Copyright 2026 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

// vmHost represents a VM host in MAAS.
type vmHost struct {
	controller *controller

	resourceURI string

	id    int
	name  string
	type_ string
	zone  *zone
	pool  *pool
}

// ID implements VmHost.
func (p *vmHost) ID() int {
	return p.id
}

// Name implements VmHost.
func (p *vmHost) Name() string {
	return p.name
}

// Type implements VmHost.
func (p *vmHost) Type() string {
	return p.type_
}

// Zone implements VmHost.
func (p *vmHost) Zone() Zone {
	if p.zone == nil {
		return nil
	}
	return p.zone
}

// Pool implements VmHost.
func (p *vmHost) Pool() Pool {
	if p.pool == nil {
		return nil
	}
	return p.pool
}

// ComposeMachineArgs holds the arguments for composing a machine in a VM host.
type ComposeMachineArgs struct {
	// Hostname is the desired hostname for the composed machine (optional).
	Hostname string

	// MinCPUCount is the minimum number of CPU cores (optional).
	MinCPUCount int

	// MinMemory is the minimum RAM in MiB (optional).
	MinMemory int

	// Storage is the list of storage specs for the machine (optional).
	// The first entry is treated as the root disk by MAAS.
	Storage []StorageSpec

	// Interfaces is the list of network interface specs (optional).
	Interfaces []InterfaceSpec

	// Zone is the desired zone name (optional).
	Zone string

	// Pool is the desired pool name (optional).
	Pool string
}

// ComposeMachine implements VmHost.
func (p *vmHost) ComposeMachine(args ComposeMachineArgs) (Machine, error) {
	return p.controller.ComposeMachine(p.id, args)
}

func (a *ComposeMachineArgs) storage() string {
	var values []string
	for _, spec := range a.Storage {
		values = append(values, spec.String())
	}
	return strings.Join(values, ",")
}

func (a *ComposeMachineArgs) interfaces() string {
	var values []string
	for _, spec := range a.Interfaces {
		values = append(values, spec.String())
	}
	return strings.Join(values, ";")
}

func readVmHosts(apiVersion version.Number, source any) ([]*vmHost, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "vm host schema check failed")
	}
	valid := coerced.([]any)

	var deserialisationVersion version.Number
	for v := range vmHostDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(apiVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no vm host read func for version %s", apiVersion)
	}
	readFunc := vmHostDeserializationFuncs[deserialisationVersion]

	var result []*vmHost
	for i, value := range valid {
		src, ok := value.(map[string]any)
		if !ok {
			return nil, errors.Errorf("unexpected value for vm host %d, %T", i, value)
		}
		p, err := readFunc(src)
		if err != nil {
			return nil, errors.Annotatef(err, "vm host %d", i)
		}
		result = append(result, p)
	}
	return result, nil
}

type vmHostDeserializationFunc func(map[string]any) (*vmHost, error)

var vmHostDeserializationFuncs = map[version.Number]vmHostDeserializationFunc{
	twoDotOh: vmHost_2_0,
}

func vmHost_2_0(source map[string]any) (*vmHost, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"id":           schema.ForceInt(),
		"name":         schema.String(),
		"type":         schema.String(),
		"zone":         schema.StringMap(schema.Any()),
		"pool":         schema.StringMap(schema.Any()),
	}
	defaults := schema.Defaults{
		"zone": schema.Omit,
		"pool": schema.Omit,
		"type": "",
	}

	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "vm host 2.0 schema check failed")
	}
	valid := coerced.(map[string]any)

	id, err := toIntValue(valid["id"])
	if err != nil {
		return nil, errors.Annotate(err, "vm host id")
	}

	result := &vmHost{
		resourceURI: valid["resource_uri"].(string),
		id:          id,
		name:        valid["name"].(string),
		type_:       valid["type"].(string),
	}

	if zoneMap, ok := valid["zone"].(map[string]any); ok {
		z, err := zone_2_0(zoneMap)
		if err != nil {
			return nil, errors.Annotate(err, "vm host zone")
		}
		result.zone = z
	}

	if poolMap, ok := valid["pool"].(map[string]any); ok {
		p, err := pool_2_0(poolMap)
		if err != nil {
			return nil, errors.Annotate(err, "vm host pool")
		}
		result.pool = p
	}

	return result, nil
}

func toIntValue(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// VmHosts implements Controller.
// It returns the list of VM hosts known to the MAAS controller.
func (c *controller) VmHosts() ([]VmHost, error) {
	source, err := c.getVmHosts()
	if err != nil {
		return nil, err
	}
	vmHosts, err := readVmHosts(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []VmHost
	for _, p := range vmHosts {
		p.controller = c
		result = append(result, p)
	}
	return result, nil
}

var vmHostEndpointUnavailableError = errors.ConstError("vm-host endpoint unavailable")

func isVmHostEndpointUnavailable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err) == vmHostEndpointUnavailableError
}

func (c *controller) getVmHosts() (any, error) {
	source, err := c.maybeGetVmHosts("vm-hosts")
	if err != nil {
		if !isVmHostEndpointUnavailable(err) {
			return nil, err
		}
	} else {
		return source, err
	}

	source, err = c.maybeGetVmHosts("pods")
	if err != nil {
		if isVmHostEndpointUnavailable(err) {
			return nil, errors.New("vm-hosts/pods API not available on this MAAS controller")
		}
		return nil, err
	}
	return source, nil
}

func (c *controller) maybeGetVmHosts(path string) (any, error) {
	source, err := c.get(path)
	if err == nil {
		return source, nil
	}

	svrErr, ok := errors.Cause(err).(ServerError)
	if !ok {
		return nil, NewUnexpectedError(err)
	}

	switch svrErr.StatusCode {
	case http.StatusNotFound, http.StatusGone:
		return nil, vmHostEndpointUnavailableError
	default:
		return nil, NewUnexpectedError(err)
	}
}

// ComposeMachine implements Controller.
// It composes (creates) a new machine in the VM host specified by vmHostID.
// Returns an error that satisfies IsNoMatchError if the VM host cannot satisfy
// the requested constraints.
func (c *controller) ComposeMachine(vmHostID int, args ComposeMachineArgs) (Machine, error) {
	params := NewURLParams()
	params.MaybeAdd("hostname", args.Hostname)
	params.MaybeAddInt("cores", args.MinCPUCount)
	params.MaybeAddInt("memory", args.MinMemory)
	params.MaybeAdd("storage", args.storage())
	params.MaybeAdd("interfaces", args.interfaces())
	if args.Zone != "" {
		zoneID, err := c.zoneIDByName(args.Zone)
		if err != nil {
			return nil, errors.Trace(err)
		}
		params.MaybeAddInt("zone", zoneID)
	}
	params.MaybeAdd("pool", args.Pool)

	result, err := c.composeMachine(vmHostID, params.Values)
	if err != nil {
		return nil, err
	}

	rawMachine, ok := result.(map[string]any)
	if !ok {
		return nil, errors.Errorf("unexpected compose response type %T", result)
	}
	if wrappedMachine, ok := rawMachine["machine"].(map[string]any); ok {
		rawMachine = wrappedMachine
	}

	systemID, ok := rawMachine["system_id"].(string)
	if !ok || systemID == "" {
		return nil, errors.New("compose response missing system_id")
	}
	machineSource, err := c.get(fmt.Sprintf("machines/%s", systemID))
	if err != nil {
		return nil, errors.Trace(err)
	}
	machine, err := readMachine(c.apiVersion, machineSource)
	if err != nil {
		return nil, errors.Trace(err)
	}
	machine.controller = c
	return machine, nil
}

func (c *controller) composeMachine(vmHostID int, params url.Values) (any, error) {
	result, err := c.maybeComposeMachine(fmt.Sprintf("vm-hosts/%d", vmHostID), params)
	if err != nil {
		if !isVmHostEndpointUnavailable(err) {
			return nil, err
		}
		result, err = c.maybeComposeMachine(fmt.Sprintf("pods/%d", vmHostID), params)
		if err != nil {
			if isVmHostEndpointUnavailable(err) {
				return nil, errors.New("vm-hosts/pods API not available on this MAAS controller")
			}
			return nil, err
		}
	}
	return result, nil
}

func (c *controller) maybeComposeMachine(path string, params url.Values) (any, error) {
	result, err := c.post(path, "compose", params)
	if err == nil {
		return result, nil
	}

	svrErr, ok := errors.Cause(err).(ServerError)
	if !ok {
		return nil, NewUnexpectedError(err)
	}

	switch svrErr.StatusCode {
	case http.StatusConflict:
		return nil, errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
	case http.StatusBadRequest:
		return nil, errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
	case http.StatusNotFound, http.StatusGone:
		return nil, vmHostEndpointUnavailableError
	default:
		return nil, NewUnexpectedError(err)
	}
}
