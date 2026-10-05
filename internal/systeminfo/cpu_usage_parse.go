package systeminfo

import (
	"fmt"
	"strconv"
	"strings"
)

func parseCPUTimes(data string) (cpuTimes, error) {
	first, _, _ := strings.Cut(data, "\n")
	fields := strings.Fields(first)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, fmt.Errorf("missing aggregate CPU times")
	}
	var result cpuTimes
	// Guest time is already included in user/nice. Do not count it twice.
	for index, field := range fields[1:min(len(fields), 9)] {
		ticks, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuTimes{}, fmt.Errorf("parse CPU times: %w", err)
		}
		result.total += ticks
		if index == 3 || index == 4 {
			result.idle += ticks
		}
	}
	return result, nil
}
