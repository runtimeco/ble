package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreBluetooth

#import "bt.h"
*/
import "C"

import "unsafe"

func byteArrToByteSlice(byteArr C.struct_byte_arr) []byte {
	return C.GoBytes(unsafe.Pointer(byteArr.data), byteArr.length)
}

func byteSliceToByteArr(b []byte) C.struct_byte_arr {
	return C.struct_byte_arr{
		data:   (*C.uint8_t)(C.CBytes(b)),
		length: C.int(len(b)),
	}
}

// cArrGetAddr retrieves the *address* of an element from a C array.
func cArrGetAddr(arr unsafe.Pointer, elemSize uintptr, idx int) unsafe.Pointer {
	base := uintptr(arr)
	off := uintptr(idx) * elemSize
	cur := base + off
	return unsafe.Pointer(cur)
}

// cArrGetStr retrieves an element from a C array of strings (i.e., char**) and
// converts it to a Go string.
func cArrGetStr(arr **C.char, idx int) string {
	dummy := (*C.char)(nil)
	elemSize := unsafe.Sizeof(dummy)

	ptr := cArrGetAddr(unsafe.Pointer(arr), elemSize, idx)
	cstr := *(**C.char)(ptr)
	return C.GoString(cstr)
}
