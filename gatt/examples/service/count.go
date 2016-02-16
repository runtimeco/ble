package service

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/currantlabs/bt/gatt"
)

// NewCountService ...
func NewCountService() *gatt.Service {
	n := 0
	s := gatt.NewService(gatt.MustParseUUID("09fc95c0-c111-11e3-9904-0002a5d5c51b"))
	s.AddCharacteristic(gatt.MustParseUUID("11fac9e0-c111-11e3-9246-0002a5d5c51b")).Handle(
		gatt.CharRead,
		gatt.HandlerFunc(func(ctx context.Context, rsp *gatt.ResponseWriter) {
			fmt.Fprintf(rsp, "count: %d", n)
			n++
		}))

	s.AddCharacteristic(gatt.MustParseUUID("16fe0d80-c111-11e3-b8c8-0002a5d5c51b")).Handle(
		gatt.CharWrite,
		gatt.HandlerFunc(func(ctx context.Context, rsp *gatt.ResponseWriter) {
			data := gatt.Data(ctx)
			log.Println("Wrote:", string(data))
		}))

	s.AddCharacteristic(gatt.MustParseUUID("1c927b50-c116-11e3-8a33-0800200c9a66")).Handle(
		gatt.CharIndicate|gatt.CharNotify,
		gatt.HandlerFunc(func(ctx context.Context, rsp *gatt.ResponseWriter) {
			n := gatt.Notifier(ctx)
			cnt := 0
			for !n.Done() {
				fmt.Fprintf(n, "Count: %d", cnt)
				cnt++
				time.Sleep(time.Second)
			}
		}))

	return s
}
