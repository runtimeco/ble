package dev

import (
	"github.com/runtimeco/ble"
	"github.com/runtimeco/ble/linux"
)

// DefaultDevice ...
func DefaultDevice(opts ...ble.Option) (d ble.Device, err error) {
	return linux.NewDevice(opts...)
}
