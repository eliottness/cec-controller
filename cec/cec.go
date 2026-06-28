package cec

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Device structure
type Device struct {
	OSDName         string
	Vendor          string
	LogicalAddress  int
	ActiveSource    bool
	PowerStatus     string
	PhysicalAddress string
}

type Command struct {
	Initiator       uint32     /**< the logical address of the initiator of this message */
	Destination     uint32     /**< the logical address of the destination of this message */
	Ack             int8       /**< 1 when the ACK bit is set, 0 otherwise */
	Eom             int8       /**< 1 when the EOM bit is set, 0 otherwise */
	Opcode          int        /**< the opcode of this message */
	Parameters      DataPacket /**< the parameters attached to this message */
	OpcodeSet       int8       /**< 1 when an opcode is set, 0 otherwise (POLL message) */
	TransmitTimeout int32      /**< the timeout to use in ms */
	Operation       string
	CommandString   string
}

type DataPacket struct {
	Data interface{}
	Size int
}

type SourceActivation struct {
	LogicalAddress     int
	LogicalAddressName string
	State              bool
}

type KeyPress struct {
	KeyCode  int
	Duration int
}

// AlertType mirrors the libcec_alert enum from <libcec/cectypes.h>.
type AlertType int

const (
	AlertServiceDevice        AlertType = iota // CEC_ALERT_SERVICE_DEVICE
	AlertConnectionLost                        // CEC_ALERT_CONNECTION_LOST
	AlertPermissionError                       // CEC_ALERT_PERMISSION_ERROR
	AlertPortBusy                              // CEC_ALERT_PORT_BUSY
	AlertPhysicalAddressError                  // CEC_ALERT_PHYSICAL_ADDRESS_ERROR
	AlertTVPollFailed                          // CEC_ALERT_TV_POLL_FAILED
)

func (t AlertType) String() string {
	switch t {
	case AlertServiceDevice:
		return "ServiceDevice"
	case AlertConnectionLost:
		return "ConnectionLost"
	case AlertPermissionError:
		return "PermissionError"
	case AlertPortBusy:
		return "PortBusy"
	case AlertPhysicalAddressError:
		return "PhysicalAddressError"
	case AlertTVPollFailed:
		return "TVPollFailed"
	default:
		return fmt.Sprintf("Unknown(%d)", int(t))
	}
}

// Alert represents a libcec alert callback. The most important is
// AlertConnectionLost, fired when the adapter drops off the bus.
type Alert struct {
	Type AlertType
}

var logicalNames = []string{"TV", "Recording", "Recording2", "Tuner",
	"Playback", "Audio", "Tuner2", "Tuner3",
	"Playback2", "Recording3", "Tuner4", "Playback3",
	"Reserved", "Reserved2", "Free", "Broadcast"}

var vendorList = map[uint64]string{0x000039: "Toshiba", 0x0000F0: "Samsung",
	0x0005CD: "Denon", 0x000678: "Marantz", 0x000982: "Loewe", 0x0009B0: "Onkyo",
	0x000CB8: "Medion", 0x000CE7: "Toshiba", 0x001582: "Pulse Eight",
	0x0020C7: "Akai", 0x002467: "Aoc", 0x008045: "Panasonic", 0x00903E: "Philips",
	0x009053: "Daewoo", 0x00A0DE: "Yamaha", 0x00D0D5: "Grundig",
	0x00E036: "Pioneer", 0x00E091: "LG", 0x08001F: "Sharp", 0x080046: "Sony",
	0x18C086: "Broadcom", 0x6B746D: "Vizio", 0x8065E9: "Benq",
	0x9C645E: "Harman Kardon"}

var cecDeviceType = map[int]string{
	0: "CEC_DEVICE_TYPE_TV",
	1: "CEC_DEVICE_TYPE_RECORDING_DEVICE",
	2: "CEC_DEVICE_TYPE_RESERVED",
	3: "CEC_DEVICE_TYPE_TUNER",
	4: "CEC_DEVICE_TYPE_PLAYBACK_DEVICE",
	5: "CEC_DEVICE_TYPE_AUDIO_SYSTEM"}

var opcodes = map[int]string{
	0x82: "ACTIVE_SOURCE",
	0x04: "IMAGE_VIEW_ON",
	0x0D: "TEXT_VIEW_ON",
	0x9D: "INACTIVE_SOURCE",
	0x85: "REQUEST_ACTIVE_SOURCE",
	0x80: "ROUTING_CHANGE",
	0x81: "ROUTING_INFORMATION",
	0x86: "SET_STREAM_PATH",
	0x36: "STANDBY",
	0x0B: "RECORD_OFF",
	0x09: "RECORD_ON",
	0x0A: "RECORD_STATUS",
	0x0F: "RECORD_TV_SCREEN",
	0x33: "CLEAR_ANALOGUE_TIMER",
	0x99: "CLEAR_DIGITAL_TIMER",
	0xA1: "CLEAR_EXTERNAL_TIMER",
	0x34: "SET_ANALOGUE_TIMER",
	0x97: "SET_DIGITAL_TIMER",
	0xA2: "SET_EXTERNAL_TIMER",
	0x67: "SET_TIMER_PROGRAM_TITLE",
	0x43: "TIMER_CLEARED_STATUS",
	0x35: "TIMER_STATUS",
	0x9E: "CEC_VERSION",
	0x9F: "GET_CEC_VERSION",
	0x83: "GIVE_PHYSICAL_ADDRESS",
	0x91: "GET_MENU_LANGUAGE",
	0x84: "REPORT_PHYSICAL_ADDRESS",
	0x32: "SET_MENU_LANGUAGE",
	0x42: "DECK_CONTROL",
	0x1B: "DECK_STATUS",
	0x1A: "GIVE_DECK_STATUS",
	0x41: "PLAY",
	0x08: "GIVE_TUNER_DEVICE_STATUS",
	0x92: "SELECT_ANALOGUE_SERVICE",
	0x93: "SELECT_DIGITAL_SERVICE",
	0x07: "TUNER_DEVICE_STATUS",
	0x06: "TUNER_STEP_DECREMENT",
	0x05: "TUNER_STEP_INCREMENT",
	0x87: "DEVICE_VENDOR_ID",
	0x8C: "GIVE_DEVICE_VENDOR_ID",
	0x89: "VENDOR_COMMAND",
	0xA0: "VENDOR_COMMAND_WITH_ID",
	0x8A: "VENDOR_REMOTE_BUTTON_DOWN",
	0x8B: "VENDOR_REMOTE_BUTTON_UP",
	0x64: "SET_OSD_STRING",
	0x46: "GIVE_OSD_NAME",
	0x47: "SET_OSD_NAME",
	0x8D: "MENU_REQUEST",
	0x8E: "MENU_STATUS",
	0x44: "USER_CONTROL_PRESSED",
	0x45: "USER_CONTROL_RELEASE",
	0x8F: "GIVE_DEVICE_POWER_STATUS",
	0x90: "REPORT_POWER_STATUS",
	0x00: "FEATURE_ABORT",
	0xFF: "ABORT",
	0x71: "GIVE_AUDIO_STATUS",
	0x7D: "GIVE_SYSTEM_AUDIO_MODE_STATUS",
	0x7A: "REPORT_AUDIO_STATUS",
	0x72: "SET_SYSTEM_AUDIO_MODE",
	0x70: "SYSTEM_AUDIO_MODE_REQUEST",
	0x7E: "SYSTEM_AUDIO_MODE_STATUS",
	0x9A: "SET_AUDIO_RATE",

	/* CEC 1.4 */
	0xC0: "START_ARC",
	0xC1: "REPORT_ARC_STARTED",
	0xC2: "REPORT_ARC_ENDED",
	0xC3: "REQUEST_ARC_START",
	0xC4: "REQUEST_ARC_END",
	0xC5: "END_ARC",
	0xF8: "CDC",
	/* when this opcode is set, no opcode will be sent to the device. this is one of the reserved numbers */
	0xFD: "NONE",
}

var keyList = map[int]string{0x00: "Select", 0x01: "Up", 0x02: "Down", 0x03: "Left",
	0x04: "Right", 0x05: "RightUp", 0x06: "RightDown", 0x07: "LeftUp",
	0x08: "LeftDown", 0x09: "RootMenu", 0x0A: "SetupMenu", 0x0B: "ContentsMenu",
	0x0C: "FavoriteMenu", 0x0D: "Exit", 0x20: "0", 0x21: "1", 0x22: "2", 0x23: "3",
	0x24: "4", 0x25: "5", 0x26: "6", 0x27: "7", 0x28: "8", 0x29: "9", 0x2A: "Dot",
	0x2B: "Enter", 0x2C: "Clear", 0x2F: "NextFavorite", 0x30: "ChannelUp",
	0x31: "ChannelDown", 0x32: "PreviousChannel", 0x33: "SoundSelect",
	0x34: "InputSelect", 0x35: "DisplayInformation", 0x36: "Help",
	0x37: "PageUp", 0x38: "PageDown", 0x40: "Power", 0x41: "VolumeUp",
	0x42: "VolumeDown", 0x43: "Mute", 0x44: "Play", 0x45: "Stop", 0x46: "Pause",
	0x47: "Record", 0x48: "Rewind", 0x49: "FastForward", 0x4A: "Eject",
	0x4B: "Forward", 0x4C: "Backward", 0x4D: "StopRecord", 0x4E: "PauseRecord",
	0x50: "Angle", 0x51: "SubPicture", 0x52: "VideoOnDemand",
	0x53: "ElectronicProgramGuide", 0x54: "TimerProgramming",
	0x55: "InitialConfiguration", 0x60: "PlayFunction", 0x61: "PausePlay",
	0x62: "RecordFunction", 0x63: "PauseRecordFunction",
	0x64: "StopFunction", 0x65: "MuteFunction",
	0x66: "RestoreVolume", 0x67: "Tune", 0x68: "SelectMedia",
	0x69: "SelectAvInput", 0x6A: "SelectAudioInput", 0x6B: "PowerToggle",
	0x6C: "PowerOff", 0x6D: "PowerOn", 0x71: "Blue", 0x72: "Red", 0x73: "Green",
	0x74: "Yellow", 0x75: "F5", 0x76: "Data", 0x91: "AnReturn",
	0x96: "Max"}

// Open - open a new connection to the CEC device with the given name
func Open(name string, deviceName string) (*Connection, error) {
	c := new(Connection)

	var err error

	c.connection, err = cecInit(c, deviceName)
	if err != nil {
		slog.Error("Error initializing connection", "error", err)
		return nil, err
	}

	slog.Info("CEC initialized", "deviceName", deviceName)

	adapter, err := getAdapter(c.connection, name)
	if err != nil {
		slog.Error("Error retrieving adapter", "error", err)
		return nil, err
	}

	slog.Info("Adapter retrieved", "name", name)

	err = openAdapter(c.connection, adapter)
	if err != nil {
		slog.Error("Error opening adapter", "error", err)
		return nil, err
	}

	slog.Info("Adapter opened")

	return c, nil
}

// Key - send key press and release commands (hold key for 10ms) to the device
// at the given address, the key code can be specified as a hex-code or by
// its name
func (c *Connection) Key(address int, key interface{}) {
	var keycode int

	switch key := key.(type) {
	case string:
		if len(key) == 4 && key[:2] == "0x" {
			keybytes, err := hex.DecodeString(key[2:])
			if err != nil {
				slog.Error("Could not decode key", "error", err)
				return
			}
			keycode = int(keybytes[0])
		} else {
			keycode = GetKeyCodeByName(key)
		}
	case int:
		keycode = key
	default:
		slog.Error("Invalid key type", "keytype", key)
		return
	}
	er := c.KeyPress(address, keycode)
	if er != nil {
		slog.Error("Error handling key press", "error", er)
		return
	}
	time.Sleep(10 * time.Millisecond)
	er = c.KeyRelease(address)
	if er != nil {
		slog.Error("Error handling key release", "error", er)
		return
	}
}

func (c *Connection) commandReceived(msg *Command) {
	slog.Debug("CEC command", "opcodeIdx", msg.Opcode, "opcode", opcodes[msg.Opcode])

	if c.Commands != nil {
		c.Commands <- msg
	}
}

func (c *Connection) messageReceived(msg string) {
	if c.Messages != nil {
		c.Messages <- msg
	}
}

func (c *Connection) keyPressed(k *KeyPress) {
	slog.Debug("CEC key pressed", "key", k)

	if c.KeyPresses != nil {
		c.KeyPresses <- k
	}
}

func (c *Connection) alertReceived(a *Alert) {
	slog.Debug("CEC alert", "type", a.Type)

	if c.Alerts == nil {
		return
	}
	// This runs on a libcec callback thread; never block it. Drop the alert if
	// no consumer is draining the channel.
	select {
	case c.Alerts <- a:
	default:
		slog.Warn("CEC alert channel full, dropping alert", "type", a.Type)
	}
}

func (c *Connection) menuActivated(s bool) {
	slog.Debug("CEC menu activated", "state", s)

	if c.MenuActivations != nil {
		c.MenuActivations <- s
	}
}

func (c *Connection) sourceActivated(src *SourceActivation) {
	slog.Debug("CEC source activated",
		"logicalAddress", src.LogicalAddress,
		"logicalAddressName", src.LogicalAddressName,
		"state", src.State)

	if c.SourceActivations != nil {
		c.SourceActivations <- src
	}
}

// List - list active devices (returns a map of Devices)
func (c *Connection) List() map[string]Device {
	devices := make(map[string]Device)

	activeDevices := c.GetActiveDevices()

	for address, active := range activeDevices {
		if active {
			var dev Device

			dev.LogicalAddress = address
			dev.PhysicalAddress = c.GetDevicePhysicalAddress(address)
			dev.OSDName = c.GetDeviceOSDName(address)
			dev.PowerStatus = c.GetDevicePowerStatus(address)
			dev.ActiveSource = c.IsActiveSource(address)
			dev.Vendor = GetVendorByID(c.GetDeviceVendorID(address))

			devices[logicalNames[address]] = dev
		}
	}
	return devices
}

func (c *Connection) Ping() int {
	return pingAdapter(c.connection)
}

// removeSeparators - remove separators (":", "-", " ", "_")
func removeSeparators(in string) string {
	out := strings.Map(func(r rune) rune {
		if strings.IndexRune(":-_ ", r) < 0 {
			return r
		}
		return -1
	}, in)

	return (out)
}

// GetKeyCodeByName - get the keycode by its name
func GetKeyCodeByName(name string) int {
	name = removeSeparators(name)
	name = strings.ToLower(name)

	for code, value := range keyList {
		if strings.ToLower(value) == name {
			return code
		}
	}

	return -1
}

// GetLogicalAddressByName - get logical address by its name
func GetLogicalAddressByName(name string) int {
	name = removeSeparators(name)
	l := len(name)

	if l > 0 && name[l-1] == '1' {
		name = name[:l-1]
	}

	name = strings.ToLower(name)

	for i := 0; i < 16; i++ {
		if strings.ToLower(logicalNames[i]) == name {
			return i
		}
	}

	if name == "unregistered" {
		return 15
	}

	return -1
}

// GetLogicalNameByAddress - get logical name by address
func GetLogicalNameByAddress(addr int) string {
	return logicalNames[addr]
}

// GetVendorByID - Get vendor by ID
func GetVendorByID(id uint64) string {
	return vendorList[id]
}
