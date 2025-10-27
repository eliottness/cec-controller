package main

import (
	"testing"
)

func TestKeyMapStructure(t *testing.T) {
	// Test that KeyMap structure is properly defined
	km := &KeyMap{
		cecToLinux:       make(map[int][]int),
		volumeController: nil,
	}

	if km == nil {
		t.Fatal("Expected KeyMap instance, got nil")
	}
	if km.cecToLinux == nil {
		t.Fatal("Expected cecToLinux map to be initialized")
	}

	// Test adding a mapping
	km.cecToLinux[1] = []int{105}
	if mapping, ok := km.cecToLinux[1]; !ok || len(mapping) != 1 || mapping[0] != 105 {
		t.Error("Failed to add mapping to KeyMap")
	}
}

func TestKeyMapMapping(t *testing.T) {
	km := &KeyMap{
		cecToLinux:       make(map[int][]int),
		volumeController: nil,
	}

	// Test single key mapping
	km.cecToLinux[1] = []int{105}
	if mapping, ok := km.cecToLinux[1]; !ok || len(mapping) != 1 || mapping[0] != 105 {
		t.Error("Failed to map single key")
	}

	// Test multiple key combination
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
		volumeController: nil,
	}

	// Test lookup of mapped keys
	if _, ok := km.cecToLinux[1]; !ok {
		t.Error("Expected key 1 to be mapped")
	}

	if _, ok := km.cecToLinux[2]; !ok {
		t.Error("Expected key 2 to be mapped")
	}

	// Test lookup of unmapped key
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
		volumeController: nil,
	}

	// Test concurrent reads (should be safe)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				done <- true
			}()

			for j := 0; j < 100; j++ {
				_ = km.cecToLinux[1]
				_ = km.cecToLinux[2]
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestKeyMapWithVolumeController(t *testing.T) {
	mockVC := NewMockVolumeController(5)
	km := &KeyMap{
		cecToLinux:       make(map[int][]int),
		volumeController: mockVC,
	}

	// Test volume up
	km.handleVolumeKey("volume_up")
	vol, _ := mockVC.GetVolume()
	if vol != 55 {
		t.Errorf("Expected volume 55 after volume_up, got %d", vol)
	}

	// Test volume down
	km.handleVolumeKey("volume_down")
	vol, _ = mockVC.GetVolume()
	if vol != 50 {
		t.Errorf("Expected volume 50 after volume_down, got %d", vol)
	}

	// Test mute
	km.handleVolumeKey("mute")
	muted, _ := mockVC.IsMuted()
	if !muted {
		t.Error("Expected muted after mute toggle")
	}
}

func TestKeyMapWithoutVolumeController(t *testing.T) {
	km := &KeyMap{
		cecToLinux:       make(map[int][]int),
		volumeController: nil,
	}

	// Should not crash when volume controller is nil
	km.handleVolumeKey("volume_up")
	km.handleVolumeKey("volume_down")
	km.handleVolumeKey("mute")
}
