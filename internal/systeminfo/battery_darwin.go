//go:build darwin

package systeminfo

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var batteryPercentagePattern = regexp.MustCompile(`([0-9]+)%`)
var batteryStatusPattern = regexp.MustCompile(`(?m)[0-9]+%;\s*([^;]+);`)
var batteryRemainingPattern = regexp.MustCompile(`([0-9]+):([0-9]{2}) remaining`)

func readBattery() (Battery, error) {
	output, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return Battery{}, nil
	}

	return parseDarwinBattery(string(output))
}

func parseDarwinBattery(output string) (Battery, error) {
	match := batteryPercentagePattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return Battery{}, nil
	}
	percentage, err := strconv.Atoi(match[1])
	if err != nil {
		return Battery{}, fmt.Errorf("parse battery percentage: %w", err)
	}
	if percentage < 0 || percentage > 100 {
		return Battery{}, nil
	}

	status := ""
	if statusMatch := batteryStatusPattern.FindStringSubmatch(output); len(statusMatch) == 2 {
		status = strings.ToLower(strings.TrimSpace(statusMatch[1]))
	}

	remaining := time.Duration(0)
	remainingAvailable := false
	if status == "discharging" {
		if match := batteryRemainingPattern.FindStringSubmatch(output); len(match) == 3 {
			hours, hErr := strconv.Atoi(match[1])
			minutes, mErr := strconv.Atoi(match[2])
			if hErr == nil && mErr == nil && hours < 1000 && minutes < 60 && hours*60+minutes > 0 {
				remaining = time.Duration(hours*60+minutes) * time.Minute
				remainingAvailable = true
			}
		}
	}
	return Battery{
		Percentage:         percentage,
		Charging:           status == "charging" || status == "finishing charge",
		Available:          true,
		Source:             "pmset",
		Remaining:          remaining,
		RemainingAvailable: remainingAvailable,
	}, nil
}
