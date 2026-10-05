//go:build darwin && cgo

#import <Foundation/Foundation.h>
#include <stdint.h>

int shellux_volume_capacity(int64_t *total, int64_t *available) {
    @autoreleasepool {
        NSURL *url = [NSURL fileURLWithPath:@"/System/Volumes/Data"];
        NSDictionary *values = [url resourceValuesForKeys:@[
            NSURLVolumeTotalCapacityKey,
            NSURLVolumeAvailableCapacityForImportantUsageKey
        ] error:nil];
        NSNumber *capacity = values[NSURLVolumeTotalCapacityKey];
        NSNumber *free = values[NSURLVolumeAvailableCapacityForImportantUsageKey];
        if (capacity == nil || free == nil) return 0;
        *total = capacity.longLongValue;
        *available = free.longLongValue;
        return 1;
    }
}
