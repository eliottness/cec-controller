package main

import (
	"errors"
	"testing"

	"github.com/claes/cec"
)

// MockCECConnection is a mock implementation of CECConnection for testing.
// Semantics follow standard Go: nil = success, non-nil = failure.
type MockCECConnection struct {
	PowerOnFunc  func(address int) error
	StandbyFunc  func(address int) error
	CloseFunc    func()
	PowerOnCalls []int
	StandbyCalls []int
	CloseCalled  bool
}

func (m *MockCECConnection) PowerOn(address int) error {
	m.PowerOnCalls = append(m.PowerOnCalls, address)
	if m.PowerOnFunc != nil {
		return m.PowerOnFunc(address)
	}
	return nil // nil = success
}

func (m *MockCECConnection) Standby(address int) error {
	m.StandbyCalls = append(m.StandbyCalls, address)
	if m.StandbyFunc != nil {
		return m.StandbyFunc(address)
	}
	return nil // nil = success
}

func (m *MockCECConnection) Close() {
	m.CloseCalled = true
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

func (m *MockCECConnection) SetKeyPressesChan(chan *cec.KeyPress) {
	// No-op for mock
}

// newTestCEC creates a CEC instance with the given mock connection, bypassing cec.Open.
func newTestCEC(conn CECConnection, opener func(string, string) (CECConnection, error)) *CEC {
	if opener == nil {
		opener = func(string, string) (CECConnection, error) {
			return nil, errors.New("no opener configured")
		}
	}
	return &CEC{
		conn:       conn,
		retries:    3,
		adapter:    "test",
		deviceName: "test",
		cecOpener:  opener,
		keyPresses: make(chan *cec.KeyPress, 1),
	}
}

func TestCECConnection_Interface(t *testing.T) {
	var _ CECConnection = (*MockCECConnection)(nil)
}

func TestMockCECConnection_PowerOn(t *testing.T) {
	mock := &MockCECConnection{}

	err := mock.PowerOn(0)
	if err != nil {
		t.Errorf("Expected nil (success), got %v", err)
	}
	if len(mock.PowerOnCalls) != 1 || mock.PowerOnCalls[0] != 0 {
		t.Errorf("Expected PowerOn called with address 0, got %v", mock.PowerOnCalls)
	}
}

func TestMockCECConnection_Standby(t *testing.T) {
	mock := &MockCECConnection{}

	err := mock.Standby(1)
	if err != nil {
		t.Errorf("Expected nil (success), got %v", err)
	}
	if len(mock.StandbyCalls) != 1 || mock.StandbyCalls[0] != 1 {
		t.Errorf("Expected Standby called with address 1, got %v", mock.StandbyCalls)
	}
}

func TestMockCECConnection_Close(t *testing.T) {
	mock := &MockCECConnection{}
	if mock.CloseCalled {
		t.Error("Expected CloseCalled to be false initially")
	}
	mock.Close()
	if !mock.CloseCalled {
		t.Error("Expected CloseCalled to be true after calling Close()")
	}
}

func TestMockCECConnection_CustomFunctions(t *testing.T) {
	powerOnCalled := false
	standbyCalled := false
	closeCalled := false

	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			powerOnCalled = true
			return nil // success
		},
		StandbyFunc: func(address int) error {
			standbyCalled = true
			return errors.New("standby failed") // failure
		},
		CloseFunc: func() {
			closeCalled = true
		},
	}

	mock.PowerOn(5)
	if !powerOnCalled {
		t.Error("Expected custom PowerOnFunc to be called")
	}

	mock.Standby(6)
	if !standbyCalled {
		t.Error("Expected custom StandbyFunc to be called")
	}

	mock.Close()
	if !closeCalled {
		t.Error("Expected custom CloseFunc to be called")
	}
}

// TestCECRetryLogic tests that connectionRetries < 1 is clamped via newCECWithOpener.
func TestCECRetryLogic(t *testing.T) {
	testCases := []struct {
		input    int
		expected int
	}{
		{0, 1},
		{-1, 1},
		{1, 1},
		{5, 5},
		{10, 10},
	}

	for _, tc := range testCases {
		mock := &MockCECConnection{}
		opener := func(string, string) (CECConnection, error) {
			return mock, nil
		}
		c, err := newCECWithOpener("", "", tc.input, make(chan *cec.KeyPress, 1), opener)
		if err != nil {
			t.Fatalf("Input %d: unexpected error: %v", tc.input, err)
		}
		if c.retries != tc.expected {
			t.Errorf("Input %d: expected retries %d, got %d", tc.input, tc.expected, c.retries)
		}
	}
}

func TestCECPower_SuccessOnFirstCall(t *testing.T) {
	mock := &MockCECConnection{} // default returns nil = success
	c := newTestCEC(mock, nil)

	if err := c.PowerOn(0); err != nil {
		t.Errorf("Expected success, got %v", err)
	}
	if len(mock.PowerOnCalls) != 1 {
		t.Errorf("Expected 1 PowerOn call, got %d", len(mock.PowerOnCalls))
	}
}

func TestCECPower_ReopenOnFirstCallFailure(t *testing.T) {
	newMock := &MockCECConnection{} // returns nil = success after reopen
	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			return errors.New("connection lost") // failure triggers reopen
		},
	}
	c := newTestCEC(mock, func(string, string) (CECConnection, error) {
		return newMock, nil
	})

	if err := c.PowerOn(0); err != nil {
		t.Errorf("Expected success after reopen, got %v", err)
	}
	if len(mock.PowerOnCalls) != 1 {
		t.Errorf("Expected 1 call on original mock, got %d", len(mock.PowerOnCalls))
	}
	if len(newMock.PowerOnCalls) != 1 {
		t.Errorf("Expected 1 call on new mock after reopen, got %d", len(newMock.PowerOnCalls))
	}
}

func TestCECPower_ReopenFails(t *testing.T) {
	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			return errors.New("connection lost")
		},
	}
	c := newTestCEC(mock, func(string, string) (CECConnection, error) {
		return nil, errors.New("reopen failed")
	})

	if err := c.PowerOn(0); err == nil {
		t.Error("Expected error when reopen fails")
	}
}

func TestCECPower_SecondCallFailsAfterReopen(t *testing.T) {
	failingMock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			return errors.New("still failing after reopen")
		},
	}
	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			return errors.New("initial failure")
		},
	}
	c := newTestCEC(mock, func(string, string) (CECConnection, error) {
		return failingMock, nil
	})

	if err := c.PowerOn(0); err == nil {
		t.Error("Expected error when both calls fail")
	}
}

func TestCECStandby_SuccessOnFirstCall(t *testing.T) {
	mock := &MockCECConnection{} // default returns nil = success
	c := newTestCEC(mock, nil)

	if err := c.Standby(0); err != nil {
		t.Errorf("Expected success, got %v", err)
	}
	if len(mock.StandbyCalls) != 1 {
		t.Errorf("Expected 1 Standby call, got %d", len(mock.StandbyCalls))
	}
}

func TestCECPower_MultipleAddresses(t *testing.T) {
	mock := &MockCECConnection{} // returns nil = success

	c := newTestCEC(mock, nil)
	if err := c.PowerOn(0, 1, 2); err != nil {
		t.Errorf("Expected success for multiple addresses, got %v", err)
	}
	if len(mock.PowerOnCalls) != 3 {
		t.Errorf("Expected 3 PowerOn calls, got %d", len(mock.PowerOnCalls))
	}
}

func TestCECPower_FailureDetected(t *testing.T) {
	callCount := 0
	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			callCount++
			if callCount == 1 {
				return errors.New("failure") // non-nil = failure
			}
			return nil // nil = success
		},
	}

	// First call fails — check mock works as expected
	err := mock.PowerOn(0)
	if err == nil {
		t.Error("Expected non-nil (failure) on first call")
	}

	// Second call succeeds
	err = mock.PowerOn(0)
	if err != nil {
		t.Errorf("Expected nil (success) on second call, got %v", err)
	}
}

func TestCECConnectionWrapper_InterfaceCompliance(t *testing.T) {
	mock := &MockCECConnection{}
	var conn CECConnection = mock
	if err := conn.PowerOn(0); err != nil {
		t.Errorf("Expected nil from mock PowerOn, got %v", err)
	}
}
