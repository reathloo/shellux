//go:build darwin && cgo

package systeminfo

/*
#include <mach/mach.h>
#include <stdint.h>

static int shellux_cpu_times(uint64_t *total, uint64_t *idle) {
    host_cpu_load_info_data_t info;
    mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;
    mach_port_t host = mach_host_self();
    kern_return_t result = host_statistics(host, HOST_CPU_LOAD_INFO,
        (host_info_t)&info, &count);
    mach_port_deallocate(mach_task_self(), host);
    if (result != KERN_SUCCESS) return result;
    *total = 0;
    for (int i = 0; i < CPU_STATE_MAX; i++) *total += info.cpu_ticks[i];
    *idle = info.cpu_ticks[CPU_STATE_IDLE];
    return 0;
}
*/
import "C"

import "fmt"

func readCPUTimes() (cpuTimes, error) {
	var total, idle C.uint64_t
	if status := C.shellux_cpu_times(&total, &idle); status != 0 {
		return cpuTimes{}, fmt.Errorf("read CPU times: Mach error %d", status)
	}
	return cpuTimes{total: uint64(total), idle: uint64(idle)}, nil
}
