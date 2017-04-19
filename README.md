# ble [![GoDoc](https://godoc.org/github.com/currantlabs/ble?status.svg)](https://godoc.org/github.com/currantlabs/ble) [![Build Status](https://travis-ci.org/moogle19/ble.svg?branch=master)](https://travis-ci.org/moogle19/ble) [![Go Report Card](https://goreportcard.com/badge/github.com/currantlabs/ble)](https://goreportcard.com/report/github.com/currantlabs/ble)

### Warning: Library is still under development and may change without warning

**ble** is a [*Bluetooth Low Energy*](https://en.wikipedia.org/wiki/Bluetooth_low_energy) library for [*Go*](https://golang.org/).

Supported operating systems are *Linux* and *macOS*.


## Examples:

### Advertise a service
```go
// Create a new device
d, err := linux.NewDevice() // for macOS: darwin.NewDevice()
if err != nil {
	// handle error
	log.Fatalf(err.Error())
}
// Set the created device as default device
ble.SetDefaultDevice(d)

// Create characteristic which counter up on read
counter := 0
charUUID := ble.MustParse("00010000-0002-1000-8000-00805F9B34FB")
char := ble.NewCharacteristic(charUUID)
char.HandleRead(ble.ReadHandlerFunc(func(req ble.Request, rsp ble.ResponseWriter) {
	fmt.Fprintf(rsp, "count: Read %d", counter++)
}))

// Create a service
serviceUUID := ble.MustParse("00010000-0001-1000-8000-00805F9B34FB")
service := ble.NewService(serviceUUID)
service.AddCharacteristic(char)

// Add service to default device
err = ble.AddService(service)
if err != nil {
	// handle error
	log.Fatalf(err.Error())
}

ctx := ble.WithSigHandler(context.WithCancel(context.Background()))

// Advertise the device and service
fmt.Printf("advertising started\n")
err = ble.AdvertiseNameAndServices(ctx, "Gopher", service.UUID)
if err != nil {
	switch errors.Cause(err) {
	case context.Canceled:
		fmt.Printf("advertising canceled\n")
	default:
		log.Fatalf(err.Error())
	}
}
```

### Scan for devices
```go
// Create a new device
d, err := linux.NewDevice() // for macOS: darwin.NewDevice()
if err != nil {
	// handle error
	log.Fatalf(err.Error())
}

// Set the created device as default device
ble.SetDefaultDevice(d)

ctx := ble.WithSigHandler(context.WithCancel(context.Background()))

// Scan for devices
err = ble.Scan(
	ctx,
	true,
	func(a ble.Advertisement) {
		// Print scanned devices
		fmt.Printf("Addr: [%s] RSSI: %3d", a.Address(), a.RSSI())
		if len(a.LocalName()) > 0 {
			fmt.Printf(" Name: %s", a.LocalName())
		}
		fmt.Printf("\n")
	},
	nil,
)

if err != nil {
	switch errors.Cause(err) {
	case context.Canceled:
		fmt.Printf("canceled scanning\n")
	default:
		log.Fatalf(err.Error())
	}
}

```