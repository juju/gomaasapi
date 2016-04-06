// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

// Can't use interface as a type, so add an underscore. Yay.
type interface_ struct {
	resourceURI string

	id      int
	name    string
	type_   string
	enabled bool

	vlan  *vlan
	links []*link

	macAddress   string
	effectiveMTU int
	params       string

	parents  []string
	children []string
}

// ID implements Interface.
func (i *interface_) ID() int {
	return i.id
}

// Name implements Interface.
func (i *interface_) Name() string {
	return i.name
}

// Parents implements Interface.
func (i *interface_) Parents() []string {
	return i.parents
}

// Children implements Interface.
func (i *interface_) Children() []string {
	return i.children
}

// Type implements Interface.
func (i *interface_) Type() string {
	return i.type_
}

// Enabled implements Interface.
func (i *interface_) Enabled() bool {
	return i.enabled
}

// VLAN implements Interface.
func (i *interface_) VLAN() VLAN {
	return i.vlan
}

// Links implements Interface.
func (i *interface_) Links() []Link {
	result := make([]Link, len(i.links))
	for i, link := range i.links {
		result[i] = link
	}
	return result
}

// MACAddress implements Interface.
func (i *interface_) MACAddress() string {
	return i.macAddress
}

// EffectiveMTU implements Interface.
func (i *interface_) EffectiveMTU() int {
	return i.effectiveMTU
}

// Params implements Interface.
func (i *interface_) Params() string {
	return i.params
}

func readInterface(controllerVersion version.Number, source interface{}) (*interface_, error) {
	readFunc, err := getInterfaceDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.StringMap(schema.Any())
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "interface base schema check failed")
	}
	valid := coerced.(map[string]interface{})
	return readFunc(valid)
}

func readInterfaces(controllerVersion version.Number, source interface{}) ([]*interface_, error) {
	readFunc, err := getInterfaceDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "interface base schema check failed")
	}
	valid := coerced.([]interface{})
	return readInterfaceList(valid, readFunc)
}

func getInterfaceDeserializationFunc(controllerVersion version.Number) (interfaceDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range interfaceDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no interface read func for version %s", controllerVersion)
	}
	return interfaceDeserializationFuncs[deserialisationVersion], nil
}

func readInterfaceList(sourceList []interface{}, readFunc interfaceDeserializationFunc) ([]*interface_, error) {
	result := make([]*interface_, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for interface %d, %T", i, value)
		}
		read, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "interface %d", i)
		}
		result = append(result, read)
	}
	return result, nil
}

type interfaceDeserializationFunc func(map[string]interface{}) (*interface_, error)

var interfaceDeserializationFuncs = map[version.Number]interfaceDeserializationFunc{
	twoDotOh: interface_2_0,
}

func interface_2_0(source map[string]interface{}) (*interface_, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"id":      schema.ForceInt(),
		"name":    schema.String(),
		"type":    schema.String(),
		"enabled": schema.Bool(),

		"vlan":  schema.StringMap(schema.Any()),
		"links": schema.List(schema.StringMap(schema.Any())),

		"mac_address":   schema.String(),
		"effective_mtu": schema.ForceInt(),
		"params":        schema.String(),

		"parents":  schema.List(schema.String()),
		"children": schema.List(schema.String()),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "interface 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	vlan, err := vlan_2_0(valid["vlan"].(map[string]interface{}))
	if err != nil {
		return nil, errors.Trace(err)
	}
	links, err := readLinkList(valid["links"].([]interface{}), link_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	result := &interface_{
		resourceURI: valid["resource_uri"].(string),

		id:      valid["id"].(int),
		name:    valid["name"].(string),
		type_:   valid["type"].(string),
		enabled: valid["enabled"].(bool),

		vlan:  vlan,
		links: links,

		macAddress:   valid["mac_address"].(string),
		effectiveMTU: valid["effective_mtu"].(int),
		params:       valid["params"].(string),

		parents:  convertToStringSlice(valid["parents"]),
		children: convertToStringSlice(valid["children"]),
	}
	return result, nil
}
