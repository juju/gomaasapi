// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type space struct {
	// Add the controller in when we need to do things with the space.
	// controller Controller

	resourceURI string

	id        int
	name      string
	classType string

	vlans []*vlan
}

// Id implements Space.
func (f *space) ID() int {
	return f.id
}

// Name implements Space.
func (f *space) Name() string {
	return f.name
}

// Name implements Space.
func (f *space) ClassType() string {
	return f.classType
}

// VLANs implements Space.
func (f *space) VLANs() []VLAN {
	var result []VLAN
	for _, v := range f.vlans {
		result = append(result, v)
	}
	return result
}

func readSpaces(controllerVersion version.Number, source interface{}) ([]*space, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "space base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range spaceDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no space read func for version %s", controllerVersion)
	}
	readFunc := spaceDeserializationFuncs[deserialisationVersion]
	return readSpaceList(valid, readFunc)
}

// readSpaceList expects the values of the sourceList to be string maps.
func readSpaceList(sourceList []interface{}, readFunc spaceDeserializationFunc) ([]*space, error) {
	result := make([]*space, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for space %d, %T", i, value)
		}
		space, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "space %d", i)
		}
		result = append(result, space)
	}
	return result, nil
}

type spaceDeserializationFunc func(map[string]interface{}) (*space, error)

var spaceDeserializationFuncs = map[version.Number]spaceDeserializationFunc{
	twoDotOh: space_2_0,
}

func space_2_0(source map[string]interface{}) (*space, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"id":           schema.ForceInt(),
		"name":         schema.String(),
		"class_type":   schema.OneOf(schema.Nil(""), schema.String()),
		"vlans":        schema.List(schema.StringMap(schema.Any())),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "space 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	vlans, err := readVLANList(valid["vlans"].([]interface{}), vlan_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Since the class_type is optional, we use the two part cast assignment. If
	// the cast fails, then we get the default value we care about, which is the
	// empty string.
	classType, _ := valid["class_type"].(string)

	result := &space{
		resourceURI: valid["resource_uri"].(string),
		id:          valid["id"].(int),
		name:        valid["name"].(string),
		classType:   classType,
		vlans:       vlans,
	}
	return result, nil
}
