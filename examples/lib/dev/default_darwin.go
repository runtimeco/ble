package dev

import (
	"github.com/runtimeco/ble"
	"github.com/runtimeco/ble/darwin"
)

// DefaultDevice ...
func DefaultDevice(opts ...ble.Option) (d ble.Device, err error) {
	return darwin.NewDevice(opts...)
}
