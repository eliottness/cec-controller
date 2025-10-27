package main

import (
	"fmt"
	"testing"
)

// MockVolumeController is a mock implementation for testing
type MockVolumeController struct {
	volume int
	muted  bool
	step   int
}

func NewMockVolumeController(step int) *MockVolumeController {
	return &MockVolumeController{
		volume: 50,
		muted:  false,
		step:   step,
	}
}

func (m *MockVolumeController) VolumeUp() error {
	m.volume += m.step
	if m.volume > 100 {
		m.volume = 100
	}
	return nil
}

func (m *MockVolumeController) VolumeDown() error {
	m.volume -= m.step
	if m.volume < 0 {
		m.volume = 0
	}
	return nil
}

func (m *MockVolumeController) Mute() error {
	m.muted = !m.muted
	return nil
}

func (m *MockVolumeController) SetVolume(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("invalid volume percentage: %d", percent)
	}
	m.volume = percent
	return nil
}

func (m *MockVolumeController) GetVolume() (int, error) {
	return m.volume, nil
}

func (m *MockVolumeController) IsMuted() (bool, error) {
	return m.muted, nil
}

func TestMockVolumeController_VolumeUp(t *testing.T) {
	vc := NewMockVolumeController(5)
	
	// Start at 50%, increase by 5%
	if err := vc.VolumeUp(); err != nil {
		t.Fatalf("VolumeUp failed: %v", err)
	}
	
	vol, err := vc.GetVolume()
	if err != nil {
		t.Fatalf("GetVolume failed: %v", err)
	}
	
	if vol != 55 {
		t.Errorf("Expected volume 55, got %d", vol)
	}
}

func TestMockVolumeController_VolumeDown(t *testing.T) {
	vc := NewMockVolumeController(5)
	
	// Start at 50%, decrease by 5%
	if err := vc.VolumeDown(); err != nil {
		t.Fatalf("VolumeDown failed: %v", err)
	}
	
	vol, err := vc.GetVolume()
	if err != nil {
		t.Fatalf("GetVolume failed: %v", err)
	}
	
	if vol != 45 {
		t.Errorf("Expected volume 45, got %d", vol)
	}
}

func TestMockVolumeController_VolumeMaxLimit(t *testing.T) {
	vc := NewMockVolumeController(10)
	vc.SetVolume(95)
	
	// Try to increase beyond 100%
	vc.VolumeUp()
	vc.VolumeUp() // Should cap at 100
	
	vol, _ := vc.GetVolume()
	if vol != 100 {
		t.Errorf("Expected volume capped at 100, got %d", vol)
	}
}

func TestMockVolumeController_VolumeMinLimit(t *testing.T) {
	vc := NewMockVolumeController(10)
	vc.SetVolume(5)
	
	// Try to decrease below 0%
	vc.VolumeDown()
	vc.VolumeDown() // Should cap at 0
	
	vol, _ := vc.GetVolume()
	if vol != 0 {
		t.Errorf("Expected volume capped at 0, got %d", vol)
	}
}

func TestMockVolumeController_Mute(t *testing.T) {
	vc := NewMockVolumeController(5)
	
	// Initially not muted
	muted, err := vc.IsMuted()
	if err != nil {
		t.Fatalf("IsMuted failed: %v", err)
	}
	if muted {
		t.Error("Expected not muted initially")
	}
	
	// Toggle mute on
	if err := vc.Mute(); err != nil {
		t.Fatalf("Mute failed: %v", err)
	}
	
	muted, err = vc.IsMuted()
	if err != nil {
		t.Fatalf("IsMuted failed: %v", err)
	}
	if !muted {
		t.Error("Expected muted after first toggle")
	}
	
	// Toggle mute off
	if err := vc.Mute(); err != nil {
		t.Fatalf("Mute failed: %v", err)
	}
	
	muted, err = vc.IsMuted()
	if err != nil {
		t.Fatalf("IsMuted failed: %v", err)
	}
	if muted {
		t.Error("Expected not muted after second toggle")
	}
}

func TestMockVolumeController_SetVolume(t *testing.T) {
	vc := NewMockVolumeController(5)
	
	testCases := []struct {
		name     string
		volume   int
		expected int
	}{
		{"Set to 25%", 25, 25},
		{"Set to 75%", 75, 75},
		{"Set to 0%", 0, 0},
		{"Set to 100%", 100, 100},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := vc.SetVolume(tc.volume); err != nil {
				t.Fatalf("SetVolume failed: %v", err)
			}
			
			vol, err := vc.GetVolume()
			if err != nil {
				t.Fatalf("GetVolume failed: %v", err)
			}
			
			if vol != tc.expected {
				t.Errorf("Expected volume %d, got %d", tc.expected, vol)
			}
		})
	}
}

func TestNewVolumeController_InvalidStep(t *testing.T) {
	// Test with 0 step - should default to 5
	vc := NewVolumeController(0)
	paVC, ok := vc.(*PulseAudioVolumeController)
	if !ok {
		t.Fatal("Expected PulseAudioVolumeController")
	}
	if paVC.step != 5 {
		t.Errorf("Expected default step of 5, got %d", paVC.step)
	}
	
	// Test with > 100 step - should default to 5
	vc = NewVolumeController(150)
	paVC, ok = vc.(*PulseAudioVolumeController)
	if !ok {
		t.Fatal("Expected PulseAudioVolumeController")
	}
	if paVC.step != 5 {
		t.Errorf("Expected default step of 5, got %d", paVC.step)
	}
}

func TestNewVolumeController_ValidStep(t *testing.T) {
	vc := NewVolumeController(10)
	paVC, ok := vc.(*PulseAudioVolumeController)
	if !ok {
		t.Fatal("Expected PulseAudioVolumeController")
	}
	if paVC.step != 10 {
		t.Errorf("Expected step of 10, got %d", paVC.step)
	}
}
