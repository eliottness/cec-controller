package main

import (
	"errors"
	"testing"

	"github.com/claes/cec"
)

// MockKeyboardEmitter records Emit calls for testing.
type MockKeyboardEmitter struct {
	EmitFunc  func(keyCodes []int) error
	EmitCalls [][]int
}

func (m *MockKeyboardEmitter) Emit(keyCodes []int) error {
	m.EmitCalls = append(m.EmitCalls, keyCodes)
	if m.EmitFunc != nil {
		return m.EmitFunc(keyCodes)
	}
	return nil
}

func TestKeyMapStructure(t *testing.T) {
	km := &KeyMap{
		cecToLinux: make(map[int][]int),
	}
	if km.cecToLinux == nil {
		t.Fatal("Expected cecToLinux map to be initialized")
	}
	km.cecToLinux[1] = []int{105}
	if mapping, ok := km.cecToLinux[1]; !ok || len(mapping) != 1 || mapping[0] != 105 {
		t.Error("Failed to add mapping to KeyMap")
	}
}

func TestKeyMapMapping(t *testing.T) {
	km := &KeyMap{
		cecToLinux: make(map[int][]int),
	}
	km.cecToLinux[1] = []int{105}
	if mapping, ok := km.cecToLinux[1]; !ok || len(mapping) != 1 || mapping[0] != 105 {
		t.Error("Failed to map single key")
	}

	km.cecToLinux[2] = []int{29, 3}
	if mapping, ok := km.cecToLinux[2]; !ok || len(mapping) != 2 {
		t.Error("Failed to map key combination")
	}
}

func TestKeyMapLookup(t *testing.T) {
	km := &KeyMap{
		cecToLinux: map[int][]int{
			1: {105},
			2: {29, 3},
			3: {56, 29, 4},
		},
	}

	if _, ok := km.cecToLinux[1]; !ok {
		t.Error("Expected key 1 to be mapped")
	}
	if _, ok := km.cecToLinux[2]; !ok {
		t.Error("Expected key 2 to be mapped")
	}
	if _, ok := km.cecToLinux[999]; ok {
		t.Error("Did not expect key 999 to be mapped")
	}
}

func TestKeyMapConcurrentRead(t *testing.T) {
	km := &KeyMap{
		cecToLinux: map[int][]int{
			1: {105},
			2: {29, 3},
		},
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_ = km.cecToLinux[1]
				_ = km.cecToLinux[2]
			}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestOnKeyPress_MappedKey(t *testing.T) {
	mock := &MockKeyboardEmitter{}
	km, err := newKeyMapWithEmitter(nil, mock)
	if err != nil {
		t.Fatalf("newKeyMapWithEmitter failed: %v", err)
	}

	cecCode := cec.GetKeyCodeByName("Select")
	if cecCode == -1 {
		t.Fatal("CEC key 'Select' not found")
	}
	km.OnKeyPress(cecCode)

	if len(mock.EmitCalls) != 1 {
		t.Errorf("Expected 1 Emit call, got %d", len(mock.EmitCalls))
	}
}

func TestOnKeyPress_UnmappedKey(t *testing.T) {
	mock := &MockKeyboardEmitter{}
	km, err := newKeyMapWithEmitter(nil, mock)
	if err != nil {
		t.Fatalf("newKeyMapWithEmitter failed: %v", err)
	}

	km.OnKeyPress(99999) // definitely unmapped

	if len(mock.EmitCalls) != 0 {
		t.Errorf("Expected no Emit calls for unmapped key, got %d", len(mock.EmitCalls))
	}
}

func TestOnKeyPress_EmitterError(t *testing.T) {
	mock := &MockKeyboardEmitter{
		EmitFunc: func(keyCodes []int) error {
			return errors.New("emit failed")
		},
	}
	km, err := newKeyMapWithEmitter(nil, mock)
	if err != nil {
		t.Fatalf("newKeyMapWithEmitter failed: %v", err)
	}

	cecCode := cec.GetKeyCodeByName("Select")
	// Should not panic; error is logged internally
	km.OnKeyPress(cecCode)

	if len(mock.EmitCalls) != 1 {
		t.Errorf("Expected Emit to be called once, got %d", len(mock.EmitCalls))
	}
}

func TestOnKeyPress_Override(t *testing.T) {
	mock := &MockKeyboardEmitter{}
	overrides := map[string][]int{
		"Select": {29, 105}, // override Select to Ctrl+KP1
	}
	km, err := newKeyMapWithEmitter(overrides, mock)
	if err != nil {
		t.Fatalf("newKeyMapWithEmitter failed: %v", err)
	}

	cecCode := cec.GetKeyCodeByName("Select")
	km.OnKeyPress(cecCode)

	if len(mock.EmitCalls) != 1 {
		t.Fatalf("Expected 1 Emit call, got %d", len(mock.EmitCalls))
	}
	if len(mock.EmitCalls[0]) != 2 || mock.EmitCalls[0][0] != 29 || mock.EmitCalls[0][1] != 105 {
		t.Errorf("Expected override codes [29, 105], got %v", mock.EmitCalls[0])
	}
}
