// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type machine struct {
	// Add the controller in when we need to do things with the machine.
	// controller Controller
	resourceURI string

	systemID string
	hostname string
	fqdn     string

	operatingSystem string
	distroSeries    string
	architecture    string
	memory          int
	cpuCount        int

	ipAddresses []string
	powerState  string

	// NOTE: consider some form of status struct
	statusName    string
	statusMessage string

	zone *zone
}

// SystemId implements Machine.
func (m *machine) SystemId() string {
	return m.systemID
}

// Hostname implements Machine.
func (m *machine) Hostname() string {
	return m.hostname
}

// FQDN implements Machine.
func (m *machine) FQDN() string {
	return m.fqdn
}

// IPAddresses implements Machine.
func (m *machine) IPAddresses() []string {
	return m.ipAddresses
}

// Memory implements Machine.
func (m *machine) Memory() int {
	return m.memory
}

// CpuCount implements Machine.
func (m *machine) CpuCount() int {
	return m.cpuCount
}

// PowerState implements Machine.
func (m *machine) PowerState() string {
	return m.powerState
}

// Zone implements Machine.
func (m *machine) Zone() Zone {
	return m.zone
}

// OperatingSystem implements Machine.
func (m *machine) OperatingSystem() string {
	return m.operatingSystem
}

// DistroSeries implements Machine.
func (m *machine) DistroSeries() string {
	return m.distroSeries
}

// Architecture implements Machine.
func (m *machine) Architecture() string {
	return m.architecture
}

// StatusName implements Machine.
func (m *machine) StatusName() string {
	return m.statusName
}

// StatusMessage implements Machine.
func (m *machine) StatusMessage() string {
	return m.statusMessage
}

func readMachines(controllerVersion version.Number, source interface{}) ([]*machine, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "machine base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range machineDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no machine read func for version %s", controllerVersion)
	}
	readFunc := machineDeserializationFuncs[deserialisationVersion]
	return readMachineList(valid, readFunc)
}

func readMachineList(sourceList []interface{}, readFunc machineDeserializationFunc) ([]*machine, error) {
	result := make([]*machine, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for machine %d, %T", i, value)
		}
		machine, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "machine %d", i)
		}
		result = append(result, machine)
	}
	return result, nil
}

type machineDeserializationFunc func(map[string]interface{}) (*machine, error)

var machineDeserializationFuncs = map[version.Number]machineDeserializationFunc{
	twoDotOh: machine_2_0,
}

func machine_2_0(source map[string]interface{}) (*machine, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"system_id": schema.String(),
		"hostname":  schema.String(),
		"fqdn":      schema.String(),

		"osystem":       schema.String(),
		"distro_series": schema.String(),
		"architecture":  schema.String(),
		"memory":        schema.ForceInt(),
		"cpu_count":     schema.ForceInt(),

		"ip_addresses":   schema.List(schema.String()),
		"power_state":    schema.String(),
		"status_name":    schema.String(),
		"status_message": schema.String(),

		"zone": schema.StringMap(schema.Any()),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "machine 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	zone, err := zone_2_0(valid["zone"].(map[string]interface{}))
	if err != nil {
		return nil, errors.Trace(err)
	}

	result := &machine{
		resourceURI: valid["resource_uri"].(string),

		systemID: valid["system_id"].(string),
		hostname: valid["hostname"].(string),
		fqdn:     valid["fqdn"].(string),

		operatingSystem: valid["osystem"].(string),
		distroSeries:    valid["distro_series"].(string),
		architecture:    valid["architecture"].(string),
		memory:          valid["memory"].(int),
		cpuCount:        valid["cpu_count"].(int),

		ipAddresses:   convertToStringSlice(valid["ip_addresses"]),
		powerState:    valid["power_state"].(string),
		statusName:    valid["status_name"].(string),
		statusMessage: valid["status_message"].(string),

		zone: zone,
	}

	return result, nil
}

func convertToStringSlice(field interface{}) []string {
	if field == nil {
		return nil
	}
	fieldSlice := field.([]interface{})
	result := make([]string, len(fieldSlice))
	for i, value := range fieldSlice {
		result[i] = value.(string)
	}
	return result
}
