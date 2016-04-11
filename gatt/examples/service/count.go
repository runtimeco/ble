package service

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

// NewCountService ...
func NewCountService() *gatt.Service {
	n := 0
	s := gatt.NewService(uuid.MustParse("09fc95c0-c111-11e3-9904-0002a5d5c51b"))
	s.AddCharacteristic(uuid.MustParse("11fac9e0-c111-11e3-9246-0002a5d5c51b")).Handle(
		gatt.CharRead,
		gatt.HandlerFunc(func(ctx context.Context, rsp *gatt.ResponseWriter) {
			fmt.Fprintf(rsp, "count: %d", n)
			n++
		}))

	s.AddCharacteristic(uuid.MustParse("16fe0d80-c111-11e3-b8c8-0002a5d5c51b")).Handle(
		gatt.CharWrite,
		gatt.HandlerFunc(func(ctx context.Context, rsp *gatt.ResponseWriter) {
			data := gatt.Data(ctx)
			log.Println("Wrote:", string(data))
		}))

	s.AddCharacteristic(uuid.MustParse("1c927b50-c116-11e3-8a33-0800200c9a66")).Handle(
		gatt.CharIndicate|gatt.CharNotify,
		gatt.HandlerFunc(func(ctx context.Context, rsp *gatt.ResponseWriter) {
			n := gatt.Notifier(ctx)
			cnt := 0
			log.Printf("Subscribed")
			for {
				select {
				case <-ctx.Done():
					log.Printf("Unsubscribed")
					return
				case <-time.After(time.Second):
					if _, err := fmt.Fprintf(n, "Count: %d", cnt); err != nil {
						log.Printf("Failed to write : %s", err)
						return
					}
					cnt++
				}
			}
		}))

	return s
}
