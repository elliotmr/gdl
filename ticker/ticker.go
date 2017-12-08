package ticker

import "time"


// TODO: make a windows version that uses GetTick
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
