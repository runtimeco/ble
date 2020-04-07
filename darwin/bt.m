#import <Foundation/Foundation.h>
#import <CoreBluetooth/CoreBluetooth.h>
#import "bt.h"

dispatch_queue_t bt_queue;
static bool bt_loop_active;

/**
 * Starts a thread that processes CoreBluetooth events from the queue.
 */
void
bt_start()
{
    if (bt_loop_active) {
        return;
    }
    bt_loop_active = true;

    dispatch_async(dispatch_get_main_queue(), ^{
        NSRunLoop *rl;
        rl = [NSRunLoop currentRunLoop];

        bool done;
        do {
            done = [rl runMode:NSDefaultRunLoopMode
                    beforeDate:[NSDate distantFuture]];
        } while (bt_loop_active && !done);
    });
}

void
bt_stop()
{
    bt_loop_active = false;
}

void
bt_init()
{
    // XXX: I have no idea why a separate queue is required here.  When I
    // attempted to use the default queue, the run loop did not receive any
    // events.
    if (bt_queue == NULL) {
        bt_queue = dispatch_queue_create("bt_queue", NULL);
    }
}
