package bytesconv

import (
	"unsafe"
)

// StringToBytes converts string to byte slice without a memory allocation.
// This uses Go 1.20+ unsafe functions for zero-copy conversion.
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToString converts byte slice to string without a memory allocation.
// This uses Go 1.20+ unsafe functions for zero-copy conversion.
func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
