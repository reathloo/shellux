//go:build darwin && cgo

package systeminfo

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <IOKit/IOKitLib.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdint.h>

static int shellux_disk_counters(uint64_t *readBytes, uint64_t *writeBytes) {
    io_iterator_t iterator = IO_OBJECT_NULL;
    if (IOServiceGetMatchingServices(kIOMainPortDefault,
            IOServiceMatching("IOBlockStorageDriver"), &iterator) != KERN_SUCCESS) return 0;
    int found = 0;
    io_service_t service;
    while ((service = IOIteratorNext(iterator)) != IO_OBJECT_NULL) {
        CFTypeRef stats = IORegistryEntryCreateCFProperty(service, CFSTR("Statistics"),
            kCFAllocatorDefault, 0);
        if (stats && CFGetTypeID(stats) == CFDictionaryGetTypeID()) {
            CFDictionaryRef dictionary = (CFDictionaryRef)stats;
            CFTypeRef readValue = CFDictionaryGetValue(dictionary, CFSTR("Bytes (Read)"));
            CFTypeRef writeValue = CFDictionaryGetValue(dictionary, CFSTR("Bytes (Write)"));
            int64_t read = 0, write = 0;
            if (readValue && writeValue && CFGetTypeID(readValue) == CFNumberGetTypeID() &&
                    CFGetTypeID(writeValue) == CFNumberGetTypeID() &&
                    CFNumberGetValue((CFNumberRef)readValue, kCFNumberSInt64Type, &read) &&
                    CFNumberGetValue((CFNumberRef)writeValue, kCFNumberSInt64Type, &write) &&
                    read >= 0 && write >= 0) {
                *readBytes += (uint64_t)read;
                *writeBytes += (uint64_t)write;
                found = 1;
            }
        }
        if (stats) CFRelease(stats);
        IOObjectRelease(service);
    }
    IOObjectRelease(iterator);
    return found;
}
*/
import "C"

import "fmt"

func readDiskCounters() (diskCounters, error) {
	var read, write C.uint64_t
	if C.shellux_disk_counters(&read, &write) == 0 {
		return diskCounters{}, fmt.Errorf("macOS disk I/O counters unavailable")
	}
	return diskCounters{uint64(read), uint64(write)}, nil
}
