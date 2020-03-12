// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type machine struct {
	controller *controller

	resourceURI string

	systemID  string
	hostname  string
	fqdn      string
	owner     string
	tags      []string
	ownerData map[string]string

	operatingSystem string
	distroSeries    string
	hweKernel       string
	architecture    string
	memory          int
	cpuCount        int

	ipAddresses []string
	powerState  string

	// NOTE: consider some form of status struct
	statusName    string
	statusMessage string

	bootInterface *interface_
	interfaceSet  []*interface_
	zone          *zone
	pool          *pool
	// Don't really know the difference between these two lists:
	physicalBlockDevices []*blockdevice
	blockDevices         []*blockdevice
}

func (m *machine) updateFrom(other *machine) {
	m.resourceURI = other.resourceURI
	m.systemID = other.systemID
	m.hostname = other.hostname
	m.fqdn = other.fqdn
	m.owner = other.owner
	m.operatingSystem = other.operatingSystem
	m.distroSeries = other.distroSeries
	m.architecture = other.architecture
	m.memory = other.memory
	m.cpuCount = other.cpuCount
	m.ipAddresses = other.ipAddresses
	m.powerState = other.powerState
	m.statusName = other.statusName
	m.statusMessage = other.statusMessage
	m.zone = other.zone
	m.pool = other.pool
	m.tags = other.tags
	m.ownerData = other.ownerData
	m.physicalBlockDevices = other.physicalBlockDevices
	m.blockDevices = other.blockDevices
}

// SystemID implements Machine.
func (m *machine) SystemID() string {
	return m.systemID
}

// Hostname implements Machine.
func (m *machine) Hostname() string {
	return m.hostname
}

// FQDN implements Machine.
func (m *machine) FQDN() string {
	return m.fqdn
}

// Owner implements Machine.
func (m *machine) Owner() string {
	return m.owner
}

// Tags implements Machine.
func (m *machine) Tags() []string {
	return m.tags
}

// Pool implements Machine
func (m *machine) Pool() Pool {
	if m.pool == nil {
		return nil
	}
	return m.pool
}

// IPAddresses implements Machine.
func (m *machine) IPAddresses() []string {
	return m.ipAddresses
}

// Memory implements Machine.
func (m *machine) Memory() int {
	return m.memory
}

// CPUCount implements Machine.
func (m *machine) CPUCount() int {
	return m.cpuCount
}

// PowerState implements Machine.
func (m *machine) PowerState() string {
	return m.powerState
}

// Zone implements Machine.
func (m *machine) Zone() Zone {
	if m.zone == nil {
		return nil
	}
	return m.zone
}

// BootInterface implements Machine.
func (m *machine) BootInterface() Interface {
	if m.bootInterface == nil {
		return nil
	}
	m.bootInterface.controller = m.controller
	return m.bootInterface
}

// InterfaceSet implements Machine.
func (m *machine) InterfaceSet() []Interface {
	result := make([]Interface, len(m.interfaceSet))
	for i, v := range m.interfaceSet {
		v.controller = m.controller
		result[i] = v
	}
	return result
}

// Interface implements Machine.
func (m *machine) Interface(id int) Interface {
	for _, iface := range m.interfaceSet {
		if iface.ID() == id {
			iface.controller = m.controller
			return iface
		}
	}
	return nil
}

// OperatingSystem implements Machine.
func (m *machine) OperatingSystem() string {
	return m.operatingSystem
}

// DistroSeries implements Machine.
func (m *machine) DistroSeries() string {
	return m.distroSeries
}

// HWEKernel implements Machine.
func (m *machine) HWEKernel() string {
	return m.hweKernel
}

// Architecture implements Machine.
func (m *machine) Architecture() string {
	return m.architecture
}

// StatusName implements Machine.
func (m *machine) StatusName() string {
	return m.statusName
}

// StatusMessage implements Machine.
func (m *machine) StatusMessage() string {
	return m.statusMessage
}

// PhysicalBlockDevices implements Machine.
func (m *machine) PhysicalBlockDevices() []BlockDevice {
	result := make([]BlockDevice, len(m.physicalBlockDevices))
	for i, v := range m.physicalBlockDevices {
		v.controller = m.controller
		result[i] = v
	}
	return result
}

// PhysicalBlockDevice implements Machine.
func (m *machine) PhysicalBlockDevice(id int) BlockDevice {
	return blockDeviceById(id, m.PhysicalBlockDevices())
}

// BlockDevices implements Machine.
func (m *machine) BlockDevices() []BlockDevice {
	result := make([]BlockDevice, len(m.blockDevices))
	for i, v := range m.blockDevices {
		v.controller = m.controller
		result[i] = v
	}
	return result
}

// BlockDevice implements Machine.
func (m *machine) BlockDevice(id int) BlockDevice {
	return blockDeviceById(id, m.BlockDevices())
}

func blockDeviceById(id int, blockDevices []BlockDevice) BlockDevice {
	for _, blockDevice := range blockDevices {
		if blockDevice.ID() == id {
			return blockDevice
		}
	}
	return nil
}

// Partition implements Machine.
func (m *machine) Partition(id int) Partition {
	p := partitionById(id, m.blockDevices)
	if p != nil {
		p.controller = m.controller
	}
	return p
}

func partitionById(id int, blockDevices []*blockdevice) *partition {
	for _, blockDevice := range blockDevices {
		for _, partition := range blockDevice.partitions {
			if partition.id == id {
				return partition
			}
		}
	}
	return nil
}

// BlockDevices implements Machine (loaded dynamically)
func (m *machine) VolumeGroups() ([]VolumeGroup, error) {
	source, err := m.controller.get(m.nodesURI() + "volume-groups/")
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return nil, errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return nil, errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return nil, errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return nil, NewUnexpectedError(err)
	}

	vgs, err := readVolumeGroups(m.controller.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}

	result := make([]VolumeGroup, len(vgs))
	for i, v := range vgs {
		v.controller = m.controller
		result[i] = v
	}
	return result, nil
}

// Devices implements Machine.
func (m *machine) Devices(args DevicesArgs) ([]Device, error) {
	// Perhaps in the future, MAAS will give us a way to query just for the
	// devices for a particular parent.
	devices, err := m.controller.Devices(args)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Device
	for _, device := range devices {
		if device.Parent() == m.SystemID() {
			result = append(result, device)
		}
	}
	return result, nil
}

// UpdateMachineArgs is arguments for machine.Update
type UpdateMachineArgs struct {
	Hostname      string
	Domain        string
	PowerType     string
	PowerAddress  string
	PowerUser     string
	PowerPassword string
	PowerOpts     map[string]string
}

// Validate ensures the arguments are acceptable
func (a *UpdateMachineArgs) Validate() error {
	return nil
}

// ToParams converts arguments to URL parameters
func (a *UpdateMachineArgs) ToParams() *URLParams {
	params := NewURLParams()
	params.MaybeAdd("hostname", a.Hostname)
	params.MaybeAdd("domain", a.Domain)
	params.MaybeAdd("power_type", a.PowerType)
	params.MaybeAdd("power_parameters_power_user", a.PowerUser)
	params.MaybeAdd("power_parameters_power_password", a.PowerUser)
	params.MaybeAdd("power_parameters_power_address", a.PowerAddress)
	if a.PowerOpts != nil {
		for k, v := range a.PowerOpts {
			params.MaybeAdd(fmt.Sprintf("power_parameters_%s", k), v)
		}
	}
	return params
}

// Update implementes Machine
func (m *machine) Update(args UpdateMachineArgs) error {
	params := args.ToParams()
	result, err := m.controller.put(m.resourceURI, params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	machine, err := readMachine(m.controller.apiVersion, result)
	if err != nil {
		return errors.Trace(err)
	}
	m.updateFrom(machine)
	return nil
}

// CommissionArgs is an argument struct for Machine.Commission
type CommissionArgs struct {
	EnableSSH            bool
	SkipBMCConfig        bool
	SkipNetworking       bool
	SkipStorage          bool
	CommissioningScripts []string
	TestingScripts       []string
}

func (m *machine) Commission(args CommissionArgs) error {
	params := NewURLParams()
	params.MaybeAddBool("enableSSH", args.EnableSSH)
	params.MaybeAddBool("skip_bmc_config", args.SkipBMCConfig)
	params.MaybeAddBool("skip_networking", args.SkipNetworking)
	params.MaybeAddBool("skip_storage", args.SkipStorage)
	params.MaybeAdd("commissioning_scripts", strings.Join(args.CommissioningScripts, ","))
	params.MaybeAdd("testing_scripts", strings.Join(args.TestingScripts, ","))

	result, err := m.controller.post(m.resourceURI, "commission", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	machine, err := readMachine(m.controller.apiVersion, result)
	if err != nil {
		return errors.Trace(err)
	}
	m.updateFrom(machine)
	return nil
}

// StartArgs is an argument struct for passing parameters to the Machine.Start
// method.
type StartArgs struct {
	// UserData needs to be Base64 encoded user data for cloud-init.
	UserData     string
	DistroSeries string
	Kernel       string
	Comment      string
}

// Start implements Machine.
func (m *machine) Start(args StartArgs) error {
	params := NewURLParams()
	params.MaybeAdd("user_data", args.UserData)
	params.MaybeAdd("distro_series", args.DistroSeries)
	params.MaybeAdd("hwe_kernel", args.Kernel)
	params.MaybeAdd("comment", args.Comment)
	result, err := m.controller.post(m.resourceURI, "deploy", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	machine, err := readMachine(m.controller.apiVersion, result)
	if err != nil {
		return errors.Trace(err)
	}
	m.updateFrom(machine)
	return nil
}

// CreateMachineBondArgs is the argument structure for Machine.CreateBond
type CreateMachineBondArgs struct {
	UpdateInterfaceArgs
	Parents []Interface
}

func (a *CreateMachineBondArgs) toParams() *URLParams {
	params := a.UpdateInterfaceArgs.toParams()
	parents := []string{}
	for _, p := range a.Parents {
		parents = append(parents, fmt.Sprintf("%d", p.ID()))
	}
	params.MaybeAdd("parents", strings.Join(parents, ","))
	return params
}

// Validate ensures that all required values are non-emtpy.
func (a *CreateMachineBondArgs) Validate() error {
	return nil
}

// CreateBond implements Machine
func (m *machine) CreateBond(args CreateMachineBondArgs) (_ Interface, err error) {
	if err := args.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	params := args.toParams()
	source, err := m.controller.post(m.resourceURI+"interfaces/", "create_bond", params.Values)
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

	response, err := readInterface(m.controller.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return response, nil
}

// CreateMachineDeviceArgs is an argument structure for Machine.CreateDevice.
// Only InterfaceName and MACAddress fields are required, the others are only
// used if set. If Subnet and VLAN are both set, Subnet.VLAN() must match the
// given VLAN. On failure, returns an error satisfying errors.IsNotValid().
type CreateMachineDeviceArgs struct {
	Hostname      string
	InterfaceName string
	MACAddress    string
	Subnet        Subnet
	VLAN          VLAN
}

// Validate ensures that all required values are non-emtpy.
func (a *CreateMachineDeviceArgs) Validate() error {
	if a.InterfaceName == "" {
		return errors.NotValidf("missing InterfaceName")
	}

	if a.MACAddress == "" {
		return errors.NotValidf("missing MACAddress")
	}

	if a.Subnet != nil && a.VLAN != nil && a.Subnet.VLAN() != a.VLAN {
		msg := fmt.Sprintf(
			"given subnet %q on VLAN %d does not match given VLAN %d",
			a.Subnet.CIDR(), a.Subnet.VLAN().ID(), a.VLAN.ID(),
		)
		return errors.NewNotValid(nil, msg)
	}

	return nil
}

// CreateDevice implements Machine
func (m *machine) CreateDevice(args CreateMachineDeviceArgs) (_ Device, err error) {
	if err := args.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	device, err := m.controller.CreateDevice(CreateDeviceArgs{
		Hostname:     args.Hostname,
		MACAddresses: []string{args.MACAddress},
		Parent:       m.SystemID(),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	defer func(err *error) {
		// If there is an error return, at least try to delete the device we just created.
		if *err != nil {
			if innerErr := device.Delete(); innerErr != nil {
				logger.Warningf("could not delete device %q", device.SystemID())
			}
		}
	}(&err)

	// Update the VLAN to use for the interface, if given.
	vlanToUse := args.VLAN
	if vlanToUse == nil && args.Subnet != nil {
		vlanToUse = args.Subnet.VLAN()
	}

	// There should be one interface created for each MAC Address, and since we
	// only specified one, there should just be one response.
	interfaces := device.InterfaceSet()
	if count := len(interfaces); count != 1 {
		err := errors.Errorf("unexpected interface count for device: %d", count)
		return nil, NewUnexpectedError(err)
	}
	iface := interfaces[0]
	nameToUse := args.InterfaceName

	if err := m.updateDeviceInterface(iface, nameToUse, vlanToUse); err != nil {
		return nil, errors.Trace(err)
	}

	if args.Subnet == nil {
		// Nothing further to update.
		return device, nil
	}

	if err := m.linkDeviceInterfaceToSubnet(iface, args.Subnet); err != nil {
		return nil, errors.Trace(err)
	}

	return device, nil
}

func (m *machine) updateDeviceInterface(iface Interface, nameToUse string, vlanToUse VLAN) error {
	updateArgs := UpdateInterfaceArgs{}
	updateArgs.Name = nameToUse

	if vlanToUse != nil {
		updateArgs.VLAN = vlanToUse
	}

	if err := iface.Update(updateArgs); err != nil {
		return errors.Annotatef(err, "updating device interface %q failed", iface.Name())
	}

	return nil
}

func (m *machine) linkDeviceInterfaceToSubnet(iface Interface, subnetToUse Subnet) error {
	err := iface.LinkSubnet(LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: subnetToUse,
	})
	if err != nil {
		return errors.Annotatef(
			err, "linking device interface %q to subnet %q failed",
			iface.Name(), subnetToUse.CIDR())
	}

	return nil
}

// OwnerData implements OwnerDataHolder.
func (m *machine) OwnerData() map[string]string {
	result := make(map[string]string)
	for key, value := range m.ownerData {
		result[key] = value
	}
	return result
}

// SetOwnerData implements OwnerDataHolder.
func (m *machine) SetOwnerData(ownerData map[string]string) error {
	params := make(url.Values)
	for key, value := range ownerData {
		params.Add(key, value)
	}
	result, err := m.controller.post(m.resourceURI, "set_owner_data", params)
	if err != nil {
		return errors.Trace(err)
	}
	machine, err := readMachine(m.controller.apiVersion, result)
	if err != nil {
		return errors.Trace(err)
	}
	m.updateFrom(machine)
	return nil
}

func (m *machine) Delete() error {
	err := m.controller.delete(m.resourceURI)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func readMachine(controllerVersion version.Number, source interface{}) (*machine, error) {
	readFunc, err := getMachineDeserializationFunc(controllerVersion)
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

func readMachines(controllerVersion version.Number, source interface{}) ([]*machine, error) {
	readFunc, err := getMachineDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine base schema check failed")
	}
	valid := coerced.([]interface{})
	return readMachineList(valid, readFunc)
}

func getMachineDeserializationFunc(controllerVersion version.Number) (machineDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range machineDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no machine read func for version %s", controllerVersion)
	}
	return machineDeserializationFuncs[deserialisationVersion], nil
}

func readMachineList(sourceList []interface{}, readFunc machineDeserializationFunc) ([]*machine, error) {
	result := make([]*machine, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for machine %d, %T", i, value)
		}
		machine, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "machine %d", i)
		}
		result = append(result, machine)
	}
	return result, nil
}

type machineDeserializationFunc func(map[string]interface{}) (*machine, error)

var machineDeserializationFuncs = map[version.Number]machineDeserializationFunc{
	twoDotOh: machine_2_0,
}

func machine_2_0(source map[string]interface{}) (*machine, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"system_id":  schema.String(),
		"hostname":   schema.String(),
		"fqdn":       schema.String(),
		"owner":      schema.OneOf(schema.Nil(""), schema.String()),
		"tag_names":  schema.List(schema.String()),
		"owner_data": schema.StringMap(schema.String()),

		"osystem":       schema.String(),
		"distro_series": schema.String(),
		"hwe_kernel":    schema.OneOf(schema.Nil(""), schema.String()),
		"architecture":  schema.OneOf(schema.Nil(""), schema.String()),
		"memory":        schema.ForceInt(),
		"cpu_count":     schema.ForceInt(),

		"ip_addresses":   schema.List(schema.String()),
		"power_state":    schema.String(),
		"status_name":    schema.String(),
		"status_message": schema.OneOf(schema.Nil(""), schema.String()),

		"boot_interface": schema.OneOf(schema.Nil(""), schema.StringMap(schema.Any())),
		"interface_set":  schema.List(schema.StringMap(schema.Any())),
		"zone":           schema.StringMap(schema.Any()),
		"pool":           schema.OneOf(schema.Nil(""), schema.Any()),

		"physicalblockdevice_set": schema.List(schema.StringMap(schema.Any())),
		"blockdevice_set":         schema.List(schema.StringMap(schema.Any())),
		"volume_groups":           schema.List(schema.StringMap(schema.Any())),
	}
	defaults := schema.Defaults{
		"architecture": "",
	}

	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	var bootInterface *interface_
	if ifaceMap, ok := valid["boot_interface"].(map[string]interface{}); ok {
		bootInterface, err = interface_2_0(ifaceMap)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	interfaceSet, err := readInterfaceList(valid["interface_set"].([]interface{}), interface_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	zone, err := zone_2_0(valid["zone"].(map[string]interface{}))
	if err != nil {
		return nil, errors.Trace(err)
	}

	var pool *pool
	if valid["pool"] != nil {
		if pool, err = pool_2_0(valid["pool"].(map[string]interface{})); err != nil {
			return nil, errors.Trace(err)
		}
	}

	physicalBlockDevices, err := readBlockDeviceList(valid["physicalblockdevice_set"].([]interface{}), blockdevice_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	blockDevices, err := readBlockDeviceList(valid["blockdevice_set"].([]interface{}), blockdevice_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	architecture, _ := valid["architecture"].(string)
	statusMessage, _ := valid["status_message"].(string)
	hweKernel, _ := valid["hwe_kernel"].(string)
	owner, _ := valid["owner"].(string)
	result := &machine{
		resourceURI: valid["resource_uri"].(string),

		systemID:  valid["system_id"].(string),
		hostname:  valid["hostname"].(string),
		fqdn:      valid["fqdn"].(string),
		owner:     owner,
		tags:      convertToStringSlice(valid["tag_names"]),
		ownerData: convertToStringMap(valid["owner_data"]),

		operatingSystem: valid["osystem"].(string),
		distroSeries:    valid["distro_series"].(string),
		hweKernel:       hweKernel,
		architecture:    architecture,
		memory:          valid["memory"].(int),
		cpuCount:        valid["cpu_count"].(int),

		ipAddresses:   convertToStringSlice(valid["ip_addresses"]),
		powerState:    valid["power_state"].(string),
		statusName:    valid["status_name"].(string),
		statusMessage: statusMessage,

		bootInterface:        bootInterface,
		interfaceSet:         interfaceSet,
		zone:                 zone,
		pool:                 pool,
		physicalBlockDevices: physicalBlockDevices,
		blockDevices:         blockDevices,
	}

	return result, nil
}

func convertToStringSlice(field interface{}) []string {
	if field == nil {
		return nil
	}
	fieldSlice := field.([]interface{})
	result := make([]string, len(fieldSlice))
	for i, value := range fieldSlice {
		result[i] = value.(string)
	}
	return result
}

func convertToStringMap(field interface{}) map[string]string {
	if field == nil {
		return nil
	}
	// This function is only called after a schema Coerce, so it's
	// safe.
	fieldMap := field.(map[string]interface{})
	result := make(map[string]string)
	for key, value := range fieldMap {
		result[key] = value.(string)
	}
	return result
}

// CreateBlockDeviceArgs are required parameters
type CreateBlockDeviceArgs struct {
	Name      string // Required. Name of the block device.
	Model     string // Optional. Model of the block device.
	Serial    string // Optional. Serial number of the block device.
	IDPath    string // Optional. Only used if model and serial cannot be provided. This should be a path that is fixed and doesn't change depending on the boot order or kernel version.
	Size      int    // Required. Size of the block device.
	BlockSize int    // Required. Block size of the block device.
}

// ToParams converts arguments to URL parameters
func (a *CreateBlockDeviceArgs) toParams() *URLParams {
	params := NewURLParams()
	params.MaybeAdd("name", a.Name)
	params.MaybeAdd("model", a.Model)
	params.MaybeAdd("serial", a.Serial)
	params.MaybeAdd("id_path", a.IDPath)
	params.MaybeAddInt("size", a.Size)
	params.MaybeAddInt("block_size", a.BlockSize)
	return params
}

// Validate checks for invalid configuration
func (a *CreateBlockDeviceArgs) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name must be provided")
	}
	if a.Size <= 0 {
		return fmt.Errorf("Size must be > 0")
	}
	if a.BlockSize <= 0 {
		return fmt.Errorf("Block size must be > 0")
	}
	return nil
}

func (m *machine) nodesURI() string {
	return strings.Replace(m.resourceURI, "machines", "nodes", 1)
}

// CreateBlockDevice implementes Machine
func (m *machine) CreateBlockDevice(args CreateBlockDeviceArgs) (BlockDevice, error) {
	if err := args.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	params := args.toParams()
	source, err := m.controller.post(m.nodesURI()+"blockdevices/", "", params.Values)
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

	response, err := readBlockDevice(m.controller.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return response, nil
}

// CreateVolumeGroupArgs control creation of a volume group
type CreateVolumeGroupArgs struct {
	Name         string        // Required. Name of the volume group.
	UUID         string        // Optional. (optional) UUID of the volume group.
	BlockDevices []BlockDevice // Optional. Block devices to add to the volume group.
	Partitions   []Partition   // Optional. Partitions to add to the volume group.
}

func (a *CreateVolumeGroupArgs) toParams() *URLParams {
	params := NewURLParams()
	params.MaybeAdd("name", a.Name)
	params.MaybeAdd("uuid", a.UUID)
	if a.BlockDevices != nil {
		deviceIDs := []string{}
		for _, device := range a.BlockDevices {
			deviceIDs = append(deviceIDs, fmt.Sprintf("%d", device.ID()))
		}
		params.MaybeAdd("block_devices", strings.Join(deviceIDs, ","))
	}
	if a.Partitions != nil {
		partitionIDs := []string{}
		for _, partition := range a.Partitions {
			partitionIDs = append(partitionIDs, fmt.Sprintf("%d", partition.ID()))
		}
		params.MaybeAdd("partitions", strings.Join(partitionIDs, ","))
	}
	return params
}

// Validate checks for errors
func (a *CreateVolumeGroupArgs) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name required")
	}
	return nil
}

func (m *machine) CreateVolumeGroup(args CreateVolumeGroupArgs) (VolumeGroup, error) {
	params := args.toParams()
	source, err := m.controller.post(m.nodesURI()+"volume-groups", "", params.Values)
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

	response, err := readVolumeGroup(m.controller.apiVersion, source)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return response, nil
}
