//go:build darwin && cgo

package systeminfo

/*
#cgo LDFLAGS: -framework Foundation
#include <stdint.h>
int shellux_volume_capacity(int64_t *total, int64_t *available);
*/
import "C"

import "fmt"

func nativeVolumeInfo() (Volume, error) {
	var total, available C.int64_t
	if C.shellux_volume_capacity(&total, &available) == 0 {
		return Volume{}, fmt.Errorf("macOS volume capacity unavailable")
	}
	return volumeFromCapacity(int64(total), int64(available))
}
