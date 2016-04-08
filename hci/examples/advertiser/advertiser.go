package main

import (
	"log"

	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
)

func main() {
	h, err := hci.NewHCI(-1)
	if err != nil {
		log.Fatalf("failed to create HCI. %s", err)
	}

	rp := cmd.LESetAdvertisingParametersRP{}
	c := cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
		AdvertisingIntervalMax:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
		AdvertisingType:         0x00,      // [0x00]: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
		OwnAddressType:          0x00,      // [0x00]: public, 0x01: random
		DirectAddressType:       0x00,      // [0x00]: public, 0x01: random
		DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
		AdvertisingChannelMap:   0x7,       // [0x07] 0x01: ch37, 0x2: ch38, 0x4: ch39
		AdvertisingFilterPolicy: 0x00,
	}
	if err := h.Send(&c, &rp); err != nil {
		log.Printf("failed to send command. err %s", err)
	}
	log.Printf("%#v", rp)
}
