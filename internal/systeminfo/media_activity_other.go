//go:build !darwin || !cgo

package systeminfo

func microphoneActivity() DeviceActivity { return DeviceActivity{} }
func cameraActivity() DeviceActivity     { return DeviceActivity{} }
