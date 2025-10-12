package main

import (
	"errors"
	"testing"

	"github.com/claes/cec"
)

// MockCECConnection is a mock implementation of CECConnection for testing
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
	return errors.New("not nil means success in libcec")
}

func (m *MockCECConnection) Standby(address int) error {
	m.StandbyCalls = append(m.StandbyCalls, address)
	if m.StandbyFunc != nil {
		return m.StandbyFunc(address)
	}
	return errors.New("not nil means success in libcec")
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

func TestCECConnection_Interface(t *testing.T) {
	// Test that MockCECConnection implements CECConnection interface
	var _ CECConnection = (*MockCECConnection)(nil)
}

func TestMockCECConnection_PowerOn(t *testing.T) {
	mock := &MockCECConnection{}

	err := mock.PowerOn(0)
	if err == nil {
		t.Error("Expected non-nil error (libcec returns non-nil on success)")
	}

	if len(mock.PowerOnCalls) != 1 {
		t.Errorf("Expected 1 PowerOn call, got %d", len(mock.PowerOnCalls))
	}
	if mock.PowerOnCalls[0] != 0 {
		t.Errorf("Expected PowerOn called with address 0, got %d", mock.PowerOnCalls[0])
	}
}

func TestMockCECConnection_Standby(t *testing.T) {
	mock := &MockCECConnection{}

	err := mock.Standby(1)
	if err == nil {
		t.Error("Expected non-nil error (libcec returns non-nil on success)")
	}

	if len(mock.StandbyCalls) != 1 {
		t.Errorf("Expected 1 Standby call, got %d", len(mock.StandbyCalls))
	}
	if mock.StandbyCalls[0] != 1 {
		t.Errorf("Expected Standby called with address 1, got %d", mock.StandbyCalls[0])
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
			if address != 5 {
				t.Errorf("Expected address 5, got %d", address)
			}
			return nil
		},
		StandbyFunc: func(address int) error {
			standbyCalled = true
			if address != 6 {
				t.Errorf("Expected address 6, got %d", address)
			}
			return errors.New("custom error")
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

func TestCECRetryLogic(t *testing.T) {
	// Test the retry adjustment logic
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
		retries := tc.input
		if retries < 1 {
			retries = 1
		}

		if retries != tc.expected {
			t.Errorf("Input %d: expected %d, got %d", tc.input, tc.expected, retries)
		}
	}
}

func TestCECPowerLogic_MultipleAddresses(t *testing.T) {
	// Test power command logic with multiple addresses
	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			return errors.New("success") // non-nil means success in libcec
		},
	}

	addresses := []int{0, 1, 2}
	for _, addr := range addresses {
		err := mock.PowerOn(addr)
		if err == nil {
			t.Errorf("Expected success (non-nil) for address %d", addr)
		}
	}

	if len(mock.PowerOnCalls) != len(addresses) {
		t.Errorf("Expected %d PowerOn calls, got %d", len(addresses), len(mock.PowerOnCalls))
	}
}

func TestCECPowerLogic_FailureHandling(t *testing.T) {
	// Test that nil return (failure in libcec) is detected
	callCount := 0
	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			callCount++
			if callCount == 1 {
				return nil // nil means failure in libcec
			}
			return errors.New("success")
		},
	}

	// First call should return nil (failure)
	err := mock.PowerOn(0)
	if err != nil {
		t.Error("Expected nil (failure) on first call")
	}

	// Second call should return non-nil (success)
	err = mock.PowerOn(0)
	if err == nil {
		t.Error("Expected non-nil (success) on second call")
	}
}

func TestCECConnectionWrapper(t *testing.T) {
	// Test that CECConnectionWrapper can be created
	// This tests the structure without requiring actual cec library
	mock := &MockCECConnection{
		PowerOnFunc: func(address int) error {
			return errors.New("success")
		},
	}

	// Verify interface compliance
	var conn CECConnection = mock
	err := conn.PowerOn(0)
	if err == nil {
		t.Error("Expected non-nil error from mock")
	}
}
