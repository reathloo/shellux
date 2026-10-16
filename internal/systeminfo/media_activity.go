package systeminfo

type DeviceActivity struct {
	Active, Available bool
}

type MediaActivity struct {
	Microphone, Camera DeviceActivity
}

func MediaActivityInfo(microphone, camera bool) MediaActivity {
	var result MediaActivity
	if microphone {
		result.Microphone = microphoneActivity()
	}
	if camera {
		result.Camera = cameraActivity()
	}
	return result
}
