// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"net/http"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type blockdevice struct {
	resourceURI string
	controller  *controller

	id      int
	uuid    string
	name    string
	model   string
	idPath  string
	path    string
	usedFor string
	tags    []string

	blockSize uint64
	usedSize  uint64
	size      uint64

	filesystem *filesystem
	partitions []*partition
}

// Type implements BlockDevice
func (b *blockdevice) Type() string {
	return "blockdevice"
}

// ID implements BlockDevice.
func (b *blockdevice) ID() int {
	return b.id
}

// UUID implements BlockDevice.
func (b *blockdevice) UUID() string {
	return b.uuid
}

// Name implements BlockDevice.
func (b *blockdevice) Name() string {
	return b.name
}

// Model implements BlockDevice.
func (b *blockdevice) Model() string {
	return b.model
}

// IDPath implements BlockDevice.
func (b *blockdevice) IDPath() string {
	return b.idPath
}

// Path implements BlockDevice.
func (b *blockdevice) Path() string {
	return b.path
}

// UsedFor implements BlockDevice.
func (b *blockdevice) UsedFor() string {
	return b.usedFor
}

// Tags implements BlockDevice.
func (b *blockdevice) Tags() []string {
	return b.tags
}

// BlockSize implements BlockDevice.
func (b *blockdevice) BlockSize() uint64 {
	return b.blockSize
}

// UsedSize implements BlockDevice.
func (b *blockdevice) UsedSize() uint64 {
	return b.usedSize
}

// Size implements BlockDevice.
func (b *blockdevice) Size() uint64 {
	return b.size
}

// FileSystem implements BlockDevice.
func (b *blockdevice) FileSystem() FileSystem {
	return b.filesystem
}

// Partitions implements BlockDevice.
func (b *blockdevice) Partitions() []Partition {
	result := make([]Partition, len(b.partitions))
	for i, v := range b.partitions {
		v.controller = b.controller
		result[i] = v
	}
	return result
}

// CreatePartitionArgs options for creating partitions
type CreatePartitionArgs struct {
	Size     int    // Optional. The size of the partition. If not specified, all available space will be used.
	UUID     string //  Optional. UUID for the partition. Only used if the partition table type for the block device is GPT.
	Bootable bool   // Optional. If the partition should be marked bootable.
}

func (a *CreatePartitionArgs) toParams() *URLParams {
	params := &URLParams{}
	params.MaybeAddInt("size", a.Size)
	params.MaybeAdd("uuid", a.UUID)
	params.MaybeAddBool("bootable", a.Bootable)
	return params
}

func (b *blockdevice) CreatePartition(args CreatePartitionArgs) (Partition, error) {
	params := args.toParams()
	source, err := b.controller.post(b.resourceURI+"/partitions/", "", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return nil, errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return nil, errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			}
		}
		return nil, NewUnexpectedError(err)
	}

	response, err := readPartition(b.controller.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return response, nil
}

func getBlockDeviceDeserializationFunc(controllerVersion version.Number) (blockdeviceDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range blockdeviceDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no blockdevice read func for version %s", controllerVersion)
	}
	return blockdeviceDeserializationFuncs[deserialisationVersion], nil
}

func readBlockDevice(controllerVersion version.Number, source interface{}) (*blockdevice, error) {
	readFunc, err := getBlockDeviceDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, err
	}

	checker := schema.StringMap(schema.Any())
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine base schema check failed")
	}
	valid := coerced.(map[string]interface{})

	return readFunc(valid)
}

func readBlockDevices(controllerVersion version.Number, source interface{}) ([]*blockdevice, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "blockdevice base schema check failed")
	}
	valid := coerced.([]interface{})

	readFunc, err := getBlockDeviceDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, err
	}
	return readBlockDeviceList(valid, readFunc)
}

// readBlockDeviceList expects the values of the sourceList to be string maps.
func readBlockDeviceList(sourceList []interface{}, readFunc blockdeviceDeserializationFunc) ([]*blockdevice, error) {
	result := make([]*blockdevice, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for blockdevice %d, %T", i, value)
		}
		blockdevice, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "blockdevice %d", i)
		}
		result = append(result, blockdevice)
	}
	return result, nil
}

type blockdeviceDeserializationFunc func(map[string]interface{}) (*blockdevice, error)

var blockdeviceDeserializationFuncs = map[version.Number]blockdeviceDeserializationFunc{
	twoDotOh: blockdevice_2_0,
}

func blockdevice_2_0(source map[string]interface{}) (*blockdevice, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"id":       schema.ForceInt(),
		"uuid":     schema.OneOf(schema.Nil(""), schema.String()),
		"name":     schema.String(),
		"model":    schema.OneOf(schema.Nil(""), schema.String()),
		"id_path":  schema.OneOf(schema.Nil(""), schema.String()),
		"path":     schema.String(),
		"used_for": schema.String(),
		"tags":     schema.List(schema.String()),

		"block_size": schema.ForceUint(),
		"used_size":  schema.ForceUint(),
		"size":       schema.ForceUint(),

		"filesystem": schema.OneOf(schema.Nil(""), schema.StringMap(schema.Any())),
		"partitions": schema.List(schema.StringMap(schema.Any())),
	}
	checker := schema.FieldMap(fields, nil)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "blockdevice 2.0 schema check failed")
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
	partitions, err := readPartitionList(valid["partitions"].([]interface{}), partition_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	uuid, _ := valid["uuid"].(string)
	model, _ := valid["model"].(string)
	idPath, _ := valid["id_path"].(string)
	result := &blockdevice{
		resourceURI: valid["resource_uri"].(string),

		id:      valid["id"].(int),
		uuid:    uuid,
		name:    valid["name"].(string),
		model:   model,
		idPath:  idPath,
		path:    valid["path"].(string),
		usedFor: valid["used_for"].(string),
		tags:    convertToStringSlice(valid["tags"]),

		blockSize: valid["block_size"].(uint64),
		usedSize:  valid["used_size"].(uint64),
		size:      valid["size"].(uint64),

		filesystem: filesystem,
		partitions: partitions,
	}
	return result, nil
}
