# Testing Documentation

This document describes the unit tests for the cec-controller project.

## Overview

The project includes comprehensive unit tests for all major components with mocked interfaces to avoid dependencies on hardware (CEC adapters) and system services (D-Bus).

## Test Files

### 1. `cec_test.go`
Tests for the CEC (HDMI-CEC) component.

**Key Features:**
- Mock implementation of `CECConnection` interface
- Tests for power on/standby commands
- Tests for retry logic
- Tests for connection handling and closing

**Tests:**
- `TestCECConnection_Interface` - Verifies interface compliance
- `TestMockCECConnection_PowerOn` - Tests PowerOn command
- `TestMockCECConnection_Standby` - Tests Standby command
- `TestMockCECConnection_Close` - Tests connection closing
- `TestMockCECConnection_CustomFunctions` - Tests custom function hooks
- `TestCECRetryLogic` - Tests retry adjustment logic
- `TestCECPowerLogic_MultipleAddresses` - Tests power commands to multiple devices
- `TestCECPowerLogic_FailureHandling` - Tests failure detection and handling
- `TestCECConnectionWrapper` - Tests wrapper implementation

### 2. `power_events_test.go`
Tests for the power event monitoring component.

**Key Features:**
- Mock implementation for D-Bus signals
- Tests for systemd-logind power event handling
- Tests for sleep, resume, and shutdown events
- Context cancellation handling

**Tests:**
- `TestPowerEventType_Constants` - Verifies power event type constants
- `TestPowerEvent_Structure` - Tests PowerEvent structure
- `TestMockPowerEventListener_PrepareForSleep_Active` - Tests sleep event (going to sleep)
- `TestMockPowerEventListener_PrepareForSleep_Inactive` - Tests resume event (waking up)
- `TestMockPowerEventListener_PrepareForShutdown` - Tests shutdown event
- `TestMockPowerEventListener_NilSignal` - Tests handling of nil signals
- `TestMockPowerEventListener_EmptyBody` - Tests handling of malformed signals
- `TestMockPowerEventListener_InvalidBodyType` - Tests handling of invalid body types
- `TestMockPowerEventListener_ContextCancellation` - Tests graceful shutdown

### 3. `keymap_test.go`
Tests for the CEC to Linux key mapping component.

**Key Features:**
- Tests for key mapping structure
- Tests for concurrent access patterns
- Tests for custom key mappings

**Tests:**
- `TestKeyMapStructure` - Tests KeyMap structure initialization
- `TestKeyMapMapping` - Tests single and multi-key mappings
- `TestKeyMapLookup` - Tests key lookup operations
- `TestKeyMapConcurrentRead` - Tests thread-safe concurrent access

### 4. `queue_test.go`
Tests for the event queue component.

**Key Features:**
- Tests for event channel operations
- Tests for queue serialization
- Tests for context cancellation patterns
- Tests for temporary directory handling

**Tests:**
- `TestQueueStructure` - Tests queue structure
- `TestPowerEventChannel` - Tests power event channel operations
- `TestMultiplePowerEvents` - Tests multiple event handling
- `TestQueueItemSerialization` - Tests queue item serialization
- `TestChannelBuffering` - Tests channel buffering behavior
- `TestContextCancellationPattern` - Tests context cancellation
- `TestTemporaryDirectory` - Tests temporary directory management

## Interfaces

### `interfaces.go`
Defines interfaces to enable mocking for testing:

#### CECConnection Interface
```go
type CECConnection interface {
    PowerOn(address int) error
    Standby(address int) error
    Close()
}
```
This interface abstracts the CEC library connection, allowing tests to use mocks instead of real hardware.

#### DBusConnection Interface
```go
type DBusConnection interface {
    AddMatchSignal(options ...interface{}) error
    Signal(ch chan<- interface{})
    Close() error
}
```
This interface abstracts D-Bus connections, allowing tests to mock system power events.

## Running Tests

### Run All Tests
```bash
go test -v
```

### Run Specific Test File
```bash
go test -v -run TestCEC
go test -v -run TestPowerEvent
go test -v -run TestKeyMap
go test -v -run TestQueue
```

### Run with Coverage
```bash
go test -cover
```

### Generate Coverage Report
```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Design Principles

1. **Isolation**: Tests don't require actual hardware (CEC adapters) or system services (D-Bus)
2. **Mocking**: Interfaces allow dependency injection for testing
3. **Fast Execution**: All tests complete in under 1 second
4. **Comprehensive**: Cover main functionality paths and edge cases
5. **Maintainable**: Clear test names and well-documented mock implementations

## Mock Implementations

### MockCECConnection
Tracks method calls and allows custom behavior through function fields:
- `PowerOnFunc` - Custom PowerOn behavior
- `StandbyFunc` - Custom Standby behavior  
- `CloseFunc` - Custom Close behavior
- `PowerOnCalls` - Tracks PowerOn calls
- `StandbyCalls` - Tracks Standby calls
- `CloseCalled` - Tracks Close calls

### MockPowerEventListener
Simulates D-Bus power events without requiring actual D-Bus connection:
- Accepts D-Bus signals through a channel
- Processes signals according to the same logic as production code
- Can be controlled through context cancellation

## Prerequisites for Testing

Tests can run in any environment with Go installed. The following system dependencies are **NOT** required for testing:
- libcec (HDMI-CEC library)
- D-Bus system bus
- uinput device (virtual keyboard)

For building the production binary, you'll need:
```bash
# Debian/Ubuntu
sudo apt install libcec-dev libp8-platform-dev

# Fedora
sudo dnf install libcec-devel
```

## Continuous Integration

These tests are designed to run in CI/CD pipelines without special hardware or system configuration.

## Contributing

When adding new features:
1. Create corresponding unit tests
2. Use mocks/interfaces for external dependencies
3. Ensure tests are deterministic and fast
4. Document any complex test scenarios
5. Maintain >80% code coverage for new code
