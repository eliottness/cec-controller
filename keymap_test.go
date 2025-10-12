package main

import (
	"testing"
)

func TestKeyMapStructure(t *testing.T) {
	// Test that KeyMap structure is properly defined
	km := &KeyMap{
		cecToLinux: make(map[int][]int),
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
		cecToLinux: make(map[int][]int),
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
