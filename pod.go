// Copyright 2026 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

// pod represents a VM host in MAAS.
type pod struct {
	controller *controller

	resourceURI string

	id    int
	name  string
	type_ string
	zone  *zone
	pool  *pool
}

// ID implements Pod.
func (p *pod) ID() int {
	return p.id
}

// Name implements Pod.
func (p *pod) Name() string {
	return p.name
}

// Type implements Pod.
func (p *pod) Type() string {
	return p.type_
}

// Zone implements Pod.
func (p *pod) Zone() Zone {
	if p.zone == nil {
		return nil
	}
	return p.zone
}

// Pool implements Pod.
func (p *pod) Pool() Pool {
	if p.pool == nil {
		return nil
	}
	return p.pool
}

// ComposeMachineArgs holds the arguments for composing a machine in a pod.
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

// ComposeMachine implements Pod.
func (p *pod) ComposeMachine(args ComposeMachineArgs) (Machine, error) {
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

func readPods(apiVersion version.Number, source interface{}) ([]*pod, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "pod schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range podDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(apiVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no pod read func for version %s", apiVersion)
	}
	readFunc := podDeserializationFuncs[deserialisationVersion]

	var result []*pod
	for i, value := range valid {
		src, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for pod %d, %T", i, value)
		}
		p, err := readFunc(src)
		if err != nil {
			return nil, errors.Annotatef(err, "pod %d", i)
		}
		result = append(result, p)
	}
	return result, nil
}

type podDeserializationFunc func(map[string]interface{}) (*pod, error)

var podDeserializationFuncs = map[version.Number]podDeserializationFunc{
	twoDotOh: pod_2_0,
}

func pod_2_0(source map[string]interface{}) (*pod, error) {
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
		return nil, WrapWithDeserializationError(err, "pod 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})

	id, err := toIntValue(valid["id"])
	if err != nil {
		return nil, errors.Annotate(err, "pod id")
	}

	result := &pod{
		resourceURI: valid["resource_uri"].(string),
		id:          id,
		name:        valid["name"].(string),
		type_:       valid["type"].(string),
	}

	if zoneMap, ok := valid["zone"].(map[string]interface{}); ok {
		z, err := zone_2_0(zoneMap)
		if err != nil {
			return nil, errors.Annotate(err, "pod zone")
		}
		result.zone = z
	}

	if poolMap, ok := valid["pool"].(map[string]interface{}); ok {
		p, err := pool_2_0(poolMap)
		if err != nil {
			return nil, errors.Annotate(err, "pod pool")
		}
		result.pool = p
	}

	return result, nil
}

func toIntValue(v interface{}) (int, error) {
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

// Pods implements Controller.
// It returns the list of pods (VM hosts) known to the MAAS controller.
func (c *controller) Pods() ([]Pod, error) {
	source, err := c.get("pods")
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusNotFound {
				return nil, errors.New("pods API not available on this MAAS controller")
			}
		}
		return nil, NewUnexpectedError(err)
	}
	pods, err := readPods(c.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Pod
	for _, p := range pods {
		p.controller = c
		result = append(result, p)
	}
	return result, nil
}

// ComposeMachine implements Controller.
// It composes (creates) a new machine in the pod specified by podID.
// Returns an error that satisfies IsNoMatchError if the pod cannot satisfy
// the requested constraints.
func (c *controller) ComposeMachine(podID int, args ComposeMachineArgs) (Machine, error) {
	params := NewURLParams()
	params.MaybeAdd("hostname", args.Hostname)
	params.MaybeAddInt("cores", args.MinCPUCount)
	params.MaybeAddInt("memory", args.MinMemory)
	params.MaybeAdd("storage", args.storage())
	params.MaybeAdd("interfaces", args.interfaces())
	params.MaybeAdd("zone", args.Zone)
	params.MaybeAdd("pool", args.Pool)

	podPath := fmt.Sprintf("pods/%d", podID)
	result, err := c.post(podPath, "compose", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusConflict:
				return nil, errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			case http.StatusBadRequest:
				return nil, errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			}
		}
		return nil, NewUnexpectedError(err)
	}

	// The compose response may be a bare machine object (MAAS 2.x) or wrapped
	// as {"machine": {...}} (MAAS 3.x). In either case we only need the
	// system_id to hand back to the caller so they can allocate it.
	// Avoid using readMachine here because the compose response may omit fields
	// like "hostname" that the strict schema checker requires.
	var rawMachine map[string]interface{}
	switch v := result.(type) {
	case map[string]interface{}:
		if m, ok := v["machine"].(map[string]interface{}); ok {
			// MAAS 3.x wrapped response.
			rawMachine = m
		} else {
			// MAAS 2.x bare machine response.
			rawMachine = v
		}
	default:
		return nil, errors.Errorf("unexpected compose response type %T", result)
	}

	systemID, ok := rawMachine["system_id"].(string)
	if !ok || systemID == "" {
		return nil, errors.New("compose response missing system_id")
	}

	composed := &composedMachine{systemID: systemID, raw: rawMachine}
	return composed, nil
}

// composedMachine is a minimal Machine implementation returned by
// ComposeMachine. It carries only the system_id needed to subsequently
// allocate the machine; all other Machine methods are not meaningful at
// this stage.
type composedMachine struct {
	systemID string
	raw      map[string]interface{}
}

func (m *composedMachine) Pod() Pod                                { return nil }
func (m *composedMachine) PowerType() string                       { return "" }
func (m *composedMachine) SystemID() string                        { return m.systemID }
func (m *composedMachine) Hostname() string                        { return "" }
func (m *composedMachine) FQDN() string                            { return "" }
func (m *composedMachine) Tags() []string                          { return nil }
func (m *composedMachine) OperatingSystem() string                 { return "" }
func (m *composedMachine) DistroSeries() string                    { return "" }
func (m *composedMachine) Architecture() string                    { return "" }
func (m *composedMachine) Memory() int                             { return 0 }
func (m *composedMachine) CPUCount() int                           { return 0 }
func (m *composedMachine) HardwareInfo() map[string]string         { return nil }
func (m *composedMachine) IPAddresses() []string                   { return nil }
func (m *composedMachine) PowerState() string                      { return "" }
func (m *composedMachine) StatusName() string                      { return "" }
func (m *composedMachine) StatusMessage() string                   { return "" }
func (m *composedMachine) BootInterface() Interface                { return nil }
func (m *composedMachine) Zone() Zone                              { return nil }
func (m *composedMachine) Pool() Pool                              { return nil }
func (m *composedMachine) Start(_ StartArgs) error                 { return nil }
func (m *composedMachine) Devices(_ DevicesArgs) ([]Device, error) { return nil, nil }
func (m *composedMachine) OwnerData() map[string]string            { return nil }
func (m *composedMachine) SetOwnerData(_ map[string]string) error  { return nil }
func (m *composedMachine) CreateDevice(_ CreateMachineDeviceArgs) (Device, error) {
	return nil, errors.NotSupportedf("CreateDevice on composedMachine")
}

func (m *composedMachine) Interface(id int) Interface {
	for _, iface := range m.InterfaceSet() {
		if iface.ID() == id {
			return iface
		}
	}
	return nil
}
func (m *composedMachine) PhysicalBlockDevices() []BlockDevice { return m.BlockDevices() }
func (m *composedMachine) PhysicalBlockDevice(id int) BlockDevice {
	return m.BlockDevice(id)
}
func (m *composedMachine) BlockDevices() []BlockDevice {
	if m.raw == nil {
		return nil
	}
	vals, ok := m.raw["blockdevice_set"].([]interface{})
	if !ok {
		return nil
	}
	bds, err := readBlockDeviceList(vals, blockdevice_2_0)
	if err != nil {
		return nil
	}
	out := make([]BlockDevice, 0, len(bds))
	for _, bd := range bds {
		out = append(out, bd)
	}
	return out
}
func (m *composedMachine) BlockDevice(id int) BlockDevice {
	for _, bd := range m.BlockDevices() {
		if bd.ID() == id {
			return bd
		}
	}
	return nil
}
func (m *composedMachine) Partition(id int) Partition {
	for _, bd := range m.BlockDevices() {
		for _, p := range bd.Partitions() {
			if p.ID() == id {
				return p
			}
		}
	}
	return nil
}

func (m *composedMachine) InterfaceSet() []Interface {
	if m.raw == nil {
		return nil
	}
	vals, ok := m.raw["interface_set"].([]interface{})
	if !ok {
		return nil
	}
	var out []Interface
	for _, v := range vals {
		if ifaceMap, ok := v.(map[string]interface{}); ok {
			if iface, err := readInterface(twoDotOh, ifaceMap); err == nil {
				out = append(out, iface)
			}
		}
	}
	return out
}
