package gdl

import "github.com/pkg/errors"

const (
	InitTimer = 1 << iota
	InitAudio
	InitVideo
	InitJoystick
	InitHaptic
	InitGameController
	InitEvents
	InitNoParachute
)
const InitEverything = InitTimer | InitAudio | InitVideo | InitJoystick | InitHaptic | InitGameController | InitEvents | InitNoParachute

func Init(flags uint32) error {
	if flags & InitGameController > 0 {
		// game controller implies joystick
		flags |= InitJoystick
	}

	if flags & (InitVideo | InitJoystick) > 0 {
		// video or joystick implies event
		flags |= InitEvents
	}

	if flags & (InitHaptic | InitJoystick) > 0 {
		if err := helperWindowCreate(); err != nil {
			return errors.Wrap(err, "failed initializing joystick")
		}
	}

	if flags & InitEvents > 0 {

	}

}