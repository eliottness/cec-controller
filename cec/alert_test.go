package cec

import "testing"

func TestAlertTypeString(t *testing.T) {
	cases := map[AlertType]string{
		AlertServiceDevice:        "ServiceDevice",
		AlertConnectionLost:       "ConnectionLost",
		AlertPermissionError:      "PermissionError",
		AlertPortBusy:             "PortBusy",
		AlertPhysicalAddressError: "PhysicalAddressError",
		AlertTVPollFailed:         "TVPollFailed",
		AlertType(99):             "Unknown(99)",
	}
	for typ, want := range cases {
		if got := typ.String(); got != want {
			t.Errorf("AlertType(%d).String() = %q, want %q", int(typ), got, want)
		}
	}
}

// AlertConnectionLost must equal libcec's CEC_ALERT_CONNECTION_LOST (1); the
// recovery path keys off this value, so a mismatch would silently break it.
func TestAlertConnectionLostValue(t *testing.T) {
	if AlertConnectionLost != 1 {
		t.Errorf("AlertConnectionLost = %d, want 1 (libcec CEC_ALERT_CONNECTION_LOST)", int(AlertConnectionLost))
	}
}

// GetKeyCodeByName must be deterministic. 0x43 and 0x65 previously both mapped
// to "Mute", making the reverse lookup non-deterministic across runs.
func TestGetKeyCodeByNameMuteDeterministic(t *testing.T) {
	if got := GetKeyCodeByName("Mute"); got != 0x43 {
		t.Errorf(`GetKeyCodeByName("Mute") = %#x, want 0x43`, got)
	}
	if got := GetKeyCodeByName("mute"); got != 0x43 {
		t.Errorf(`GetKeyCodeByName("mute") = %#x, want 0x43`, got)
	}
	if got := GetKeyCodeByName("MuteFunction"); got != 0x65 {
		t.Errorf(`GetKeyCodeByName("MuteFunction") = %#x, want 0x65`, got)
	}
}

func TestGetKeyCodeByNameUnknown(t *testing.T) {
	if got := GetKeyCodeByName("NoSuchKey"); got != -1 {
		t.Errorf("GetKeyCodeByName(unknown) = %d, want -1", got)
	}
}

// GetLogicalAddressByName must not panic on empty or all-separator input.
func TestGetLogicalAddressByNameEdgeCases(t *testing.T) {
	cases := map[string]int{
		"":             -1,
		"---":          -1,
		"TV":           0,
		"unregistered": 15,
	}
	for name, want := range cases {
		if got := GetLogicalAddressByName(name); got != want {
			t.Errorf("GetLogicalAddressByName(%q) = %d, want %d", name, got, want)
		}
	}
}
