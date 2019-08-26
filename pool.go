// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type pool struct {
	// Add the controller in when we need to do things with the pool.
	// controller Controller

	resourceURI string

	name        string
	description string
}

// Name implements Pool.
func (p *pool) Name() string {
	return p.name
}

// Description implements Pool.
func (p *pool) Description() string {
	return p.description
}

func readPools(controllerVersion version.Number, source interface{}) ([]*pool, error) {
	var deserialisationVersion version.Number

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)

	if err != nil {
		return nil, errors.Annotatef(err, "pool base schema check failed")
	}

	valid := coerced.([]interface{})

	for v := range poolDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}

	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no pool read func for version %s", controllerVersion)
	}

	readFunc := poolDeserializationFuncs[deserialisationVersion]
	return readPoolList(valid, readFunc)
}

// readPoolList expects the values of the sourceList to be string maps.
func readPoolList(sourceList []interface{}, readFunc poolDeserializationFunc) ([]*pool, error) {
	result := make([]*pool, 0, len(sourceList))

	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for pool %d, %T", i, value)
		}
		pool, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "pool %d", i)
		}
		result = append(result, pool)
	}
	return result, nil
}

type poolDeserializationFunc func(map[string]interface{}) (*pool, error)

var poolDeserializationFuncs = map[version.Number]poolDeserializationFunc{
	twoDotOh: pool_2_0,
}

func pool_2_0(source map[string]interface{}) (*pool, error) {
	fields := schema.Fields{
		"name":         schema.String(),
		"description":  schema.String(),
		"resource_uri": schema.String(),
	}

	checker := schema.FieldMap(fields, nil) // no defaults

	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "pool 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	result := &pool{
		name:        valid["name"].(string),
		description: valid["description"].(string),
		resourceURI: valid["resource_uri"].(string),
	}
	return result, nil
}
