package darwin

/*
// See cutil.go for C compiler flags.
#import "bt.h"
*/
import "C"

import (
	"fmt"
	"log"
)

type BTState struct {
	Enabled bool
	Msg     string
}

var btStateCh chan BTState

func StartBTLoop() error {
	if btStateCh != nil {
		// Already running.
		return nil
	}
	btStateCh = make(chan BTState)

	C.bt_start()

	// Block until initial state change.  If Bluetooth is disabled, fail
	// immediately.
	state := <-btStateCh
	if !state.Enabled {
		StopBTLoop()
		return fmt.Errorf("failed to start CoreBluetooth: %s", state.Msg)
	}

	// Listen for and log subsequent state changes in the background.
	go func() {
		for {
			state, ok := <-btStateCh
			if !ok {
				btStateCh = nil
				return
			}

			log.Printf("CoreBluetooth state change: %s", state.Msg)
		}
	}()

	return nil
}

func StopBTLoop() {
	C.bt_stop()
	close(btStateCh)
}

func BTInit() {
	C.bt_init()
}
