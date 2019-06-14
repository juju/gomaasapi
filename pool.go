<<<<<<< HEAD
<<<<<<< HEAD
// Copyright 2019 Canonical Ltd.
=======
// Copyright 2016 Canonical Ltd.
>>>>>>> c00d3cc... Implement pools
=======
// Copyright 2019 Canonical Ltd.
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
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
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
	var deserialisationVersion version.Number

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)

	if err != nil {
		return nil, errors.Annotatef(err, "pool base schema check failed")
	}

	valid := coerced.([]interface{})

<<<<<<< HEAD
=======
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "pool base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
>>>>>>> c00d3cc... Implement pools
=======
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
	for v := range poolDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6

	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no pool read func for version %s", controllerVersion)
	}

<<<<<<< HEAD
=======
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no pool read func for version %s", controllerVersion)
	}
>>>>>>> c00d3cc... Implement pools
=======
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
	readFunc := poolDeserializationFuncs[deserialisationVersion]
	return readPoolList(valid, readFunc)
}

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
// readPoolList expects the values of the sourceList to be string maps.
func readPoolList(sourceList []interface{}, readFunc poolDeserializationFunc) ([]*pool, error) {
	result := make([]*pool, 0, len(sourceList))

<<<<<<< HEAD
=======
// readZoneList expects the values of the sourceList to be string maps.
=======
// readPoolList expects the values of the sourceList to be string maps.
>>>>>>> 925e48d... Bugfixes missed in implementing Pool.
func readPoolList(sourceList []interface{}, readFunc poolDeserializationFunc) ([]*pool, error) {
	result := make([]*pool, 0, len(sourceList))
>>>>>>> c00d3cc... Implement pools
=======
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
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
<<<<<<< HEAD
	twoDotOh: Pool2dot0,
}

<<<<<<< HEAD
func pool_2_0(source map[string]interface{}) (*pool, error) {
<<<<<<< HEAD
	fields := schema.Fields {
=======
=======
func Pool2dot0(source map[string]interface{}) (*pool, error) {
>>>>>>> 925e48d... Bugfixes missed in implementing Pool.
	fields := schema.Fields{
>>>>>>> c00d3cc... Implement pools
=======
	twoDotOh: pool_2_0,
}

func pool_2_0(source map[string]interface{}) (*pool, error) {
	fields := schema.Fields {
>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
		"name":         schema.String(),
		"description":  schema.String(),
		"resource_uri": schema.String(),
	}
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD

	checker := schema.FieldMap(fields, nil) // no defaults

=======
	checker := schema.FieldMap(fields, nil) // no defaults
>>>>>>> c00d3cc... Implement pools
=======

	checker := schema.FieldMap(fields, nil) // no defaults

>>>>>>> 925e48d... Bugfixes missed in implementing Pool.
=======

	checker := schema.FieldMap(fields, nil) // no defaults

>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
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
<<<<<<< HEAD
<<<<<<< HEAD
}

=======
}
>>>>>>> c00d3cc... Implement pools
=======
}

>>>>>>> 0d1a8ac28143a65bbc193ca10c82d68ef403d5d6
