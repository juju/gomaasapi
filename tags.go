// Copyright 2022 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version/v2"
)

type tag struct {
	resourceURI string

	name       string
	comment    string
	definition string
	kernelOpts string
}

func (tag tag) Name() string {
	return tag.name
}

func (tag tag) Comment() string {
	return tag.comment
}

func (tag tag) Definition() string {
	return tag.definition
}

func (tag tag) KernelOpts() string {
	return tag.kernelOpts
}

func readTags(controllerVersion version.Number, source interface{}) ([]*tag, error) {
	readFunc, err := getTagDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine base schema check failed")
	}

	// OK to do a direct cast here because we just coerced the interface.
	valid := coerced.([]interface{})
	return readTagList(valid, readFunc)
}

func readTagList(sourceList []interface{}, readFunc tagDeserializationFunc) ([]*tag, error) {
	result := make([]*tag, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for tag %d, %T", i, value)
		}

		machine, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "tag %d", i)
		}

		result = append(result, machine)
	}

	return result, nil
}

func getTagDeserializationFunc(controllerVersion version.Number) (tagDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range tagDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}

	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no tag read func for version %s", controllerVersion)
	}

	return tagDeserializationFuncs[deserialisationVersion], nil
}

type tagDeserializationFunc func(map[string]interface{}) (*tag, error)

var tagDeserializationFuncs = map[version.Number]tagDeserializationFunc{
	twoDotOh: tag_2_0,
}

func tag_2_0(source map[string]interface{}) (*tag, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"name":         schema.String(),
		"comment":      schema.String(),
		"definition":   schema.String(),
		"kernel_opts":  schema.String(),
	}

	defaults := schema.Defaults{
		"comment":     "",
		"definition":  "",
		"kernel_opts": "",
	}

	checker := schema.FieldMap(fields, defaults)

	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "tag 2.0 schema check failed")
	}

	valid := coerced.(map[string]interface{})

	return &tag{
		resourceURI: valid["resource_uri"].(string),
		name:        valid["name"].(string),
		comment:     valid["comment"].(string),
		definition:  valid["definition"].(string),
		kernelOpts:  valid["kernel_opts"].(string),
	}, nil
}
