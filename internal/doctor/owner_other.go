//go:build !darwin && !linux

package doctor

import "os"

func ownedByUser(os.FileInfo) bool { return false }
