package cec

// #include <libcec/cecc.h>
import "C"

import (
	"log/slog"
	"unsafe"
)

//export logMessageCallback
func logMessageCallback(c unsafe.Pointer, msg *C.cec_log_message) C.int {
	slog.Debug("CEC msg rx", "message", C.GoString(msg.message))

	conn := (*Connection)(c)
	conn.messageReceived(C.GoString(msg.message))
	return 0
}

//export keyPressed
func keyPressed(c unsafe.Pointer, code *C.cec_keypress) C.int {
	slog.Debug("CEC keycode rx", "code", code)

	conn := (*Connection)(c)
	keyPress := &KeyPress{
		KeyCode:  int(C.int(code.keycode)),
		Duration: int(code.duration),
	}
	conn.keyPressed(keyPress)
	return 0
}

//export commandReceived
func commandReceived(c unsafe.Pointer, msg *C.cec_command) C.int {
	slog.Debug("CEC command rx", "msg", msg)

	conn := (*Connection)(c)
	cmd := &Command{
		Initiator:       uint32(msg.initiator),
		Destination:     uint32(msg.destination),
		Ack:             int8(msg.ack),
		Eom:             int8(msg.eom),
		Opcode:          int(msg.opcode),
		Parameters:      DataPacket{Data: msg.parameters.data, Size: int(msg.parameters.size)},
		OpcodeSet:       int8(msg.opcode_set),
		TransmitTimeout: int32(msg.transmit_timeout),
		Operation:       opcodes[int(msg.opcode)],
		CommandString:   CreateCommandString(msg),
	}
	conn.commandReceived(cmd)

	return 0
}

//export alertReceived
func alertReceived(c unsafe.Pointer, alert_type C.libcec_alert, cec_param C.libcec_parameter) {
	// libcec's alert callback is void-returning (see the forward declaration in
	// libcec.go and cectypes.h). Forward the alert to the Connection; the
	// consumer (main loop) decides severity, so we don't log here.
	conn := (*Connection)(c)
	conn.alertReceived(&Alert{Type: AlertType(alert_type)})
}

//export sourceActivated
func sourceActivated(c unsafe.Pointer, logicalAddress C.cec_logical_address, activated int) {
	conn := (*Connection)(c)
	src := &SourceActivation{
		LogicalAddress:     int(logicalAddress),
		LogicalAddressName: GetLogicalNameByAddress(int(logicalAddress)),
		State:              activated == 1}
	conn.sourceActivated(src)
}

//export menuStateChanged
func menuStateChanged(c unsafe.Pointer, state C.cec_menu_state) C.uint8_t {
	conn := (*Connection)(c)
	// menuState is bool, 0 = activated, 1 = deactivated
	conn.menuActivated(int(state) == 0)
	return 1
}
