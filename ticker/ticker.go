package ticker

import "time"

var Start time.Time

func Initialize() {
	Start = time.Now()
}

func Get() time.Duration {
	return time.Now().Sub(Start)
}

func GetAsMS() uint32 {
	return uint32(Get() / time.Millisecond)
}
