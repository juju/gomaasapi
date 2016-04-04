// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type device struct {
	resourceURI string

	systemID string
	hostname string
	fqdn     string

	ipAddresses []string
	zone        *zone
}

// SystemID implements Device.
func (d *device) SystemID() string {
	return d.systemID
}

// Hostname implements Device.
func (d *device) Hostname() string {
	return d.hostname
}

// FQDN implements Device.
func (d *device) FQDN() string {
	return d.fqdn
}

// IPAddresses implements Device.
func (d *device) IPAddresses() []string {
	return d.ipAddresses
}

// Zone implements Device.
func (d *device) Zone() Zone {
	return d.zone
}

func readDevices(controllerVersion version.Number, source interface{}) ([]*device, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "device base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range deviceDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no device read func for version %s", controllerVersion)
	}
	readFunc := deviceDeserializationFuncs[deserialisationVersion]
	return readDeviceList(valid, readFunc)
}

// readDeviceList expects the values of the sourceList to be string maps.
func readDeviceList(sourceList []interface{}, readFunc deviceDeserializationFunc) ([]*device, error) {
	result := make([]*device, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for device %d, %T", i, value)
		}
		device, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "device %d", i)
		}
		result = append(result, device)
	}
	return result, nil
}

type deviceDeserializationFunc func(map[string]interface{}) (*device, error)

var deviceDeserializationFuncs = map[version.Number]deviceDeserializationFunc{
	twoDotOh: device_2_0,
}

func device_2_0(source map[string]interface{}) (*device, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"system_id": schema.String(),
		"hostname":  schema.String(),
		"fqdn":      schema.String(),

		"ip_addresses": schema.List(schema.String()),
		"zone":         schema.StringMap(schema.Any()),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "device 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	zone, err := zone_2_0(valid["zone"].(map[string]interface{}))
	if err != nil {
		return nil, errors.Trace(err)
	}

	result := &device{
		resourceURI: valid["resource_uri"].(string),

		systemID: valid["system_id"].(string),
		hostname: valid["hostname"].(string),
		fqdn:     valid["fqdn"].(string),

		ipAddresses: convertToStringSlice(valid["ip_addresses"]),
		zone:        zone,
	}
	return result, nil
}
