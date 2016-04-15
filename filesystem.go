// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type filesystem struct {
	fstype     string
	mountPoint string
	label      string
	uuid       string
	// no idea what the mount_options are as a value type, so ignoring for now.
}

// Type implements FileSystem.
func (f *filesystem) Type() string {
	return f.fstype
}

// MountPoint implements FileSystem.
func (f *filesystem) MountPoint() string {
	return f.mountPoint
}

// Label implements FileSystem.
func (f *filesystem) Label() string {
	return f.label
}

// UUID implements FileSystem.
func (f *filesystem) UUID() string {
	return f.uuid
}

func readFileSystems(controllerVersion version.Number, source interface{}) ([]*filesystem, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "filesystem base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range filesystemDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no filesystem read func for version %s", controllerVersion)
	}
	readFunc := filesystemDeserializationFuncs[deserialisationVersion]
	return readFileSystemList(valid, readFunc)
}

// readFileSystemList expects the values of the sourceList to be string maps.
func readFileSystemList(sourceList []interface{}, readFunc filesystemDeserializationFunc) ([]*filesystem, error) {
	result := make([]*filesystem, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for filesystem %d, %T", i, value)
		}
		filesystem, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "filesystem %d", i)
		}
		result = append(result, filesystem)
	}
	return result, nil
}

type filesystemDeserializationFunc func(map[string]interface{}) (*filesystem, error)

var filesystemDeserializationFuncs = map[version.Number]filesystemDeserializationFunc{
	twoDotOh: filesystem_2_0,
}

func filesystem_2_0(source map[string]interface{}) (*filesystem, error) {
	fields := schema.Fields{
		"fstype":      schema.String(),
		"mount_point": schema.String(),
		"label":       schema.String(),
		"uuid":        schema.String(),
		// TODO: mount_options when we know the type.
	}
	checker := schema.FieldMap(fields, nil)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "filesystem 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	result := &filesystem{
		fstype:     valid["fstype"].(string),
		mountPoint: valid["mount_point"].(string),
		label:      valid["label"].(string),
		uuid:       valid["uuid"].(string),
	}
	return result, nil
}
