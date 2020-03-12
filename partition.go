// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"net/http"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type partition struct {
	controller  *controller
	resourceURI string

	id      int
	path    string
	uuid    string
	usedFor string
	size    uint64
	tags    []string

	filesystem *filesystem
}

func (p *partition) updateFrom(other *partition) {
	p.resourceURI = other.resourceURI
	p.id = other.id
	p.path = other.path
	p.uuid = other.uuid
	p.usedFor = other.usedFor
	p.size = other.size
	p.tags = other.tags
	p.filesystem = other.filesystem
}

// Type implements Partition.
func (p *partition) Type() string {
	return "partition"
}

// ID implements Partition.
func (p *partition) ID() int {
	return p.id
}

// Path implements Partition.
func (p *partition) Path() string {
	return p.path
}

// FileSystem implements Partition.
func (p *partition) FileSystem() FileSystem {
	if p.filesystem == nil {
		return nil
	}
	return p.filesystem
}

// UUID implements Partition.
func (p *partition) UUID() string {
	return p.uuid
}

// UsedFor implements Partition.
func (p *partition) UsedFor() string {
	return p.usedFor
}

// Size implements Partition.
func (p *partition) Size() uint64 {
	return p.size
}

// Tags implements Partition.
func (p *partition) Tags() []string {
	return p.tags
}

func (p *partition) Format(args FormatStorageDeviceArgs) error {
	if err := args.Validate(); err != nil {
		return errors.Trace(err)
	}

	params := NewURLParams()
	params.MaybeAdd("fs_type", args.FSType)
	params.MaybeAdd("uuid", args.UUID)
	params.MaybeAdd("label", args.Label)

	result, err := p.controller.post(p.resourceURI, "format", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	partition, err := readPartition(p.controller.apiVersion, result)
	if err != nil {
		return errors.Trace(err)
	}
	p.updateFrom(partition)
	return nil
}

func (p *partition) Mount(args MountStorageDeviceArgs) error {
	params := args.toParams()
	source, err := p.controller.post(p.resourceURI, "mount", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	response, err := readPartition(p.controller.apiVersion, source)
	if err != nil {
		return errors.Trace(err)
	}
	p.updateFrom(response)
	return nil
}

func readPartition(controllerVersion version.Number, source interface{}) (*partition, error) {
	readFunc, err := getPartitionDeserializationFunc(controllerVersion)
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

func getPartitionDeserializationFunc(controllerVersion version.Number) (partitionDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range partitionDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no partition read func for version %s", controllerVersion)
	}
	return partitionDeserializationFuncs[deserialisationVersion], nil
}

func readPartitions(controllerVersion version.Number, source interface{}) ([]*partition, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "partition base schema check failed")
	}
	valid := coerced.([]interface{})

	readFunc, err := getPartitionDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, err
	}
	return readPartitionList(valid, readFunc)
}

// readPartitionList expects the values of the sourceList to be string maps.
func readPartitionList(sourceList []interface{}, readFunc partitionDeserializationFunc) ([]*partition, error) {
	result := make([]*partition, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for partition %d, %T", i, value)
		}
		partition, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "partition %d", i)
		}
		result = append(result, partition)
	}
	return result, nil
}

type partitionDeserializationFunc func(map[string]interface{}) (*partition, error)

var partitionDeserializationFuncs = map[version.Number]partitionDeserializationFunc{
	twoDotOh: partition_2_0,
}

func partition_2_0(source map[string]interface{}) (*partition, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"id":       schema.ForceInt(),
		"path":     schema.String(),
		"uuid":     schema.OneOf(schema.Nil(""), schema.String()),
		"used_for": schema.String(),
		"size":     schema.ForceUint(),
		"tags":     schema.List(schema.String()),

		"filesystem": schema.OneOf(schema.Nil(""), schema.StringMap(schema.Any())),
	}
	defaults := schema.Defaults{
		"tags": []string{},
	}
	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "partition 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	var filesystem *filesystem
	if fsSource, ok := valid["filesystem"].(map[string]interface{}); ok {
		if filesystem, err = filesystem2_0(fsSource); err != nil {
			return nil, errors.Trace(err)
		}
	}

	uuid, _ := valid["uuid"].(string)
	result := &partition{
		resourceURI: valid["resource_uri"].(string),

		id:      valid["id"].(int),
		path:    valid["path"].(string),
		uuid:    uuid,
		usedFor: valid["used_for"].(string),
		size:    valid["size"].(uint64),
		tags:    convertToStringSlice(valid["tags"]),

		filesystem: filesystem,
	}
	return result, nil
}
