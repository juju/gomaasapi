package gomaasapi

import (
	"net/url"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type tag struct {
	controller  *controller
	resourceURI string

	name       string
	definition string
	comment    string
}

// Name implements Tag.
func (s *tag) Name() string {
	return s.name
}

// Definition implements Tag.
func (s *tag) Definition() string {
	return s.definition
}

// Comment implements Tag.
func (s *tag) Comment() string {
	return s.definition
}

func (s *tag) Machines() ([]Machine, error) {
	source, err := s.controller.getOp(s.resourceURI, "machines")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	machines, err := readMachines(s.controller.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Machine
	for _, m := range machines {
		//m.controller = c
		result = append(result, m)
	}
	return result, nil
}

func (s *tag) AddToMachine(systemID string) error {
	params := url.Values{
		"add": []string{systemID},
	}
	_, err := s.controller.post(s.resourceURI, "update_nodes", params)
	return err
}

func (s *tag) RemoveFromMachine(systemID string) error {
	params := url.Values{
		"remove": []string{systemID},
	}
	_, err := s.controller.post(s.resourceURI, "update_nodes", params)
	return err
}

func readTags(controllerVersion version.Number, source interface{}) ([]*tag, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "tag base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range tagDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no tag read func for version %s", controllerVersion)
	}
	readFunc := tagDeserializationFuncs[deserialisationVersion]
	return readTagList(valid, readFunc)
}

func readTag(controllerVersion version.Number, source interface{}) (*tag, error) {
	var deserialisationVersion version.Number
	for v := range tagDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}

	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no tag read func for version %s", controllerVersion)
	}
	readFunc := tagDeserializationFuncs[deserialisationVersion]
	return readFunc(source.(map[string]interface{}))
}

// readTagList expects the values of the sourceList to be string maps.
func readTagList(sourceList []interface{}, readFunc tagDeserializationFunc) ([]*tag, error) {
	result := make([]*tag, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for tag %d, %T", i, value)
		}
		tag, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "tag %d", i)
		}
		result = append(result, tag)
	}
	return result, nil
}

type tagDeserializationFunc func(map[string]interface{}) (*tag, error)

var tagDeserializationFuncs = map[version.Number]tagDeserializationFunc{
	twoDotOh: tag_2_0,
}

func tag_2_0(source map[string]interface{}) (*tag, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"name":         schema.String(),
		"definition":   schema.String(),
		"comment":      schema.String(),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "tag 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	result := &tag{
		resourceURI: valid["resource_uri"].(string),
		name:        valid["name"].(string),
		comment:     valid["comment"].(string),
		definition:  valid["definition"].(string),
	}
	return result, nil
}
