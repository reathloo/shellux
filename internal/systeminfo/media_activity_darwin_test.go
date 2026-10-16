//go:build darwin && cgo

package systeminfo

import "testing"

func TestNativeMediaActivityReturnsConsistentStates(t *testing.T) {
	activity := MediaActivityInfo(true, true)
	for name, state := range map[string]DeviceActivity{"microphone": activity.Microphone, "camera": activity.Camera} {
		if state.Active && !state.Available {
			t.Fatalf("%s active without available status: %+v", name, state)
		}
		t.Logf("%s activity: available=%v active=%v", name, state.Available, state.Active)
	}
}
