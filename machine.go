// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/base64"
	"net/http"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type machine struct {
	controller *controller

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

func (m *machine) updateFrom(other *machine) {
	m.resourceURI = other.resourceURI
	m.systemID = other.systemID
	m.hostname = other.hostname
	m.fqdn = other.fqdn
	m.operatingSystem = other.operatingSystem
	m.distroSeries = other.distroSeries
	m.architecture = other.architecture
	m.memory = other.memory
	m.cpuCount = other.cpuCount
	m.ipAddresses = other.ipAddresses
	m.powerState = other.powerState
	m.statusName = other.statusName
	m.statusMessage = other.statusMessage
	m.zone = other.zone
}

// SystemID implements Machine.
func (m *machine) SystemID() string {
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

// CPUCount implements Machine.
func (m *machine) CPUCount() int {
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

// StartArgs is an argument struct for passing parameters to the Machine.Start
// method.
type StartArgs struct {
	UserData     []byte
	DistroSeries string
	Kernel       string
	Comment      string
}

// Start implements Machine.
func (m *machine) Start(args StartArgs) error {
	var encodedUserData string
	if args.UserData != nil {
		encodedUserData = base64.StdEncoding.EncodeToString(args.UserData)
	}
	params := NewURLParams()
	params.MaybeAdd("user_data", encodedUserData)
	params.MaybeAdd("distro_series", args.DistroSeries)
	params.MaybeAdd("hwe_kernel", args.Kernel)
	params.MaybeAdd("comment", args.Comment)
	result, err := m.controller.post(m.resourceURI, "deploy", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	machine, err := readMachine(m.controller.apiVersion, result)
	if err != nil {
		return errors.Trace(err)
	}
	m.updateFrom(machine)
	return nil
}

func readMachine(controllerVersion version.Number, source interface{}) (*machine, error) {
	readFunc, err := getMachineDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.StringMap(schema.Any())
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine base schema check failed")
	}
	valid := coerced.(map[string]interface{})
	return readFunc(valid)
}

func readMachines(controllerVersion version.Number, source interface{}) ([]*machine, error) {
	readFunc, err := getMachineDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine base schema check failed")
	}
	valid := coerced.([]interface{})
	return readMachineList(valid, readFunc)
}

func getMachineDeserializationFunc(controllerVersion version.Number) (machineDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range machineDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no machine read func for version %s", controllerVersion)
	}
	return machineDeserializationFuncs[deserialisationVersion], nil
}

func readMachineList(sourceList []interface{}, readFunc machineDeserializationFunc) ([]*machine, error) {
	result := make([]*machine, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for machine %d, %T", i, value)
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
		return nil, WrapWithDeserializationError(err, "machine 2.0 schema check failed")
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
