package gomaasapi

import (
	"fmt"
	"net/http"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type volumegroup struct {
	// Add the controller in when we need to do things with the zone.
	controller *controller

	resourceURI string

	id          int
	name        string
	description string
	uuid        string
	size        uint64
	devices     []*blockdevice
}

func (vg *volumegroup) Name() string {
	return vg.name
}

func (vg *volumegroup) Size() uint64 {
	return vg.size
}

func (vg *volumegroup) UUID() string {
	return vg.uuid
}

func (vg *volumegroup) Devices() []BlockDevice {
	result := make([]BlockDevice, len(vg.devices))
	for i, v := range vg.devices {
		//v.controller = d
		result[i] = v
	}
	return result
}

// CreateLogicalVolumeArgs creates a logical volume in a volume group
type CreateLogicalVolumeArgs struct {
	Name string // Required. Name of the logical volume.
	UUID string // Optional. (optional) UUID of the logical volume.
	Size int    // Required. Size of the logical volume. Must be larger than or equal to 4,194,304 bytes. E.g. 4194304.
}

// Validate ensures arguments are valid
func (a *CreateLogicalVolumeArgs) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name must be specified")
	}
	if a.Size <= 0 {
		return fmt.Errorf("A size > 0 must be specified")
	}
	return nil
}

// CreateLogicalVolume creates a logical volume in a volume group
func (vg *volumegroup) CreateLogicalVolume(args CreateLogicalVolumeArgs) (BlockDevice, error) {
	if err := args.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	params := NewURLParams()
	params.MaybeAdd("name", args.Name)
	params.MaybeAdd("uuid", args.UUID)
	params.MaybeAddInt("size", args.Size)

	result, err := vg.controller.post(vg.resourceURI, "create_logical_volume", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return nil, errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return nil, errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return nil, errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return nil, NewUnexpectedError(err)
	}

	device, err := readBlockDevice(vg.controller.apiVersion, result)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return device, nil
}

func readVolumeGroup(controllerVersion version.Number, source interface{}) (*volumegroup, error) {
	readFunc, err := getVolumeGroupDeserializationFunc(controllerVersion)
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

func getVolumeGroupDeserializationFunc(controllerVersion version.Number) (volumeGroupDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range volumeGroupDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no volumegroup read func for version %s", controllerVersion)
	}
	return volumeGroupDeserializationFuncs[deserialisationVersion], nil
}

func readVolumeGroups(controllerVersion version.Number, source interface{}) ([]*volumegroup, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "volumegroup base schema check failed")
	}
	valid := coerced.([]interface{})

	readFunc, err := getVolumeGroupDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, err
	}
	return readVolumeGroupList(valid, readFunc)
}

// readPartitionList expects the values of the sourceList to be string maps.
func readVolumeGroupList(sourceList []interface{}, readFunc volumeGroupDeserializationFunc) ([]*volumegroup, error) {
	result := make([]*volumegroup, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for volumegroup %d, %T", i, value)
		}
		volumegroup, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "volumegroup %d", i)
		}
		result = append(result, volumegroup)
	}
	return result, nil
}

type volumeGroupDeserializationFunc func(map[string]interface{}) (*volumegroup, error)

var volumeGroupDeserializationFuncs = map[version.Number]volumeGroupDeserializationFunc{
	twoDotOh: volumegroup_2_0,
}

func volumegroup_2_0(source map[string]interface{}) (*volumegroup, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"id":           schema.ForceInt(),
		"name":         schema.String(),
		"uuid":         schema.OneOf(schema.Nil(""), schema.String()),
		"size":         schema.ForceUint(),
		"devices":      schema.List(schema.StringMap(schema.Any())),
		//"used_size":            schema.String(),
		//"human_size":           schema.ForceUint(),
		//"system_id":            schema.String(),
		//"available_size":       schema.String(),
		//"logical_volumes":      schema.String(),
		//"human_used_size":      schema.String(),
		//"human_available_size": schema.String(),
	}
	defaults := schema.Defaults{
		//"tags": []string{},
	}
	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "volumegroup 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	devices, err := readBlockDeviceList(valid["devices"].([]interface{}), blockdevice_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	uuid, _ := valid["uuid"].(string)
	result := &volumegroup{
		resourceURI: valid["resource_uri"].(string),
		id:          valid["id"].(int),
		name:        valid["name"].(string),
		uuid:        uuid,
		size:        valid["size"].(uint64),
		devices:     devices,
	}
	return result, nil
}
