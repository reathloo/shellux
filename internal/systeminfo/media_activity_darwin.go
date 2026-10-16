//go:build darwin && cgo

package systeminfo

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreMediaIO -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreMediaIO/CMIOHardware.h>
#include <stdint.h>
#include <stdlib.h>

// Returns -1 when the OS cannot report activity, otherwise 0 or 1.
static int shellux_microphone_active(void) {
    AudioObjectPropertyAddress devices = {kAudioHardwarePropertyDevices,
        kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
    UInt32 size = 0;
    if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &devices, 0, NULL, &size) != noErr || !size) return -1;
    AudioDeviceID *ids = malloc(size);
    if (!ids) return -1;
    int result = -1;
    if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &devices, 0, NULL, &size, ids) == noErr) {
        for (UInt32 i = 0; i < size / sizeof(AudioDeviceID); i++) {
            AudioObjectPropertyAddress streams = {kAudioDevicePropertyStreams,
                kAudioObjectPropertyScopeInput, kAudioObjectPropertyElementMain};
            UInt32 streamSize = 0;
            if (AudioObjectGetPropertyDataSize(ids[i], &streams, 0, NULL, &streamSize) != noErr || !streamSize) continue;
            AudioObjectPropertyAddress running = {kAudioDevicePropertyDeviceIsRunningSomewhere,
                kAudioObjectPropertyScopeInput, kAudioObjectPropertyElementMain};
            UInt32 active = 0, activeSize = sizeof(active);
            if (AudioObjectGetPropertyData(ids[i], &running, 0, NULL, &activeSize, &active) != noErr) continue;
            result = 0;
            if (active) { result = 1; break; }
        }
    }
    free(ids);
    return result;
}

static int shellux_camera_active(void) {
    CMIOObjectPropertyAddress devices = {kCMIOHardwarePropertyDevices,
        kCMIOObjectPropertyScopeGlobal, kCMIOObjectPropertyElementMain};
    UInt32 size = 0;
    if (CMIOObjectGetPropertyDataSize(kCMIOObjectSystemObject, &devices, 0, NULL, &size) != noErr || !size) return -1;
    CMIODeviceID *ids = malloc(size);
    if (!ids) return -1;
    int result = -1;
    UInt32 used = 0;
    if (CMIOObjectGetPropertyData(kCMIOObjectSystemObject, &devices, 0, NULL, size, &used, ids) == noErr) {
        for (UInt32 i = 0; i < used / sizeof(CMIODeviceID); i++) {
            CMIOObjectPropertyAddress running = {kCMIODevicePropertyDeviceIsRunningSomewhere,
                kCMIOObjectPropertyScopeGlobal, kCMIOObjectPropertyElementMain};
            UInt32 active = 0, activeSize = sizeof(active);
            if (CMIOObjectGetPropertyData(ids[i], &running, 0, NULL, activeSize, &used, &active) != noErr) continue;
            result = 0;
            if (active) { result = 1; break; }
        }
    }
    free(ids);
    return result;
}
*/
import "C"

func microphoneActivity() DeviceActivity {
	active := C.shellux_microphone_active()
	return DeviceActivity{Active: active == 1, Available: active >= 0}
}

func cameraActivity() DeviceActivity {
	active := C.shellux_camera_active()
	return DeviceActivity{Active: active == 1, Available: active >= 0}
}
