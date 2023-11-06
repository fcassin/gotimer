package timer

// #cgo CFLAGS: -g -Wall
// #include <stdlib.h>
// #include "timer.h"
import "C"
import (
	"fmt"
	"time"
)

var verbose bool

func ReadOSTimer() int64 {
	return time.Now().UnixMicro()
}

func GetOSTimerFreq() int64 {
	return 1000000
}

func ReadCPUTimer() int64 {
	cvalue := C.ReadCPUTimer()
	return int64(cvalue)
}

func GetCPUTimerFreq(millisecondsToWait int64) int64 {
	osFrequency := GetOSTimerFreq()
	if verbose {
		fmt.Printf("   OS Freq: %v (reported)\n", osFrequency)
	}

	cpuStart := ReadCPUTimer()
	osStart := ReadOSTimer()
	var osEnd, osElapsed int64
	osWaitTime := osFrequency * millisecondsToWait / 1000
	for osElapsed < osWaitTime {
		osEnd = ReadOSTimer()
		osElapsed = osEnd - osStart
	}

	cpuEnd := ReadCPUTimer()
	cpuElapsed := cpuEnd - cpuStart
	cpuFrequency := osFrequency * cpuElapsed / osElapsed

	if verbose {
		fmt.Printf("  OS timer: %v -> %v = %v elapsed\n", osStart, osEnd, osElapsed)
		fmt.Printf("OS seconds: %.4f\n", float64(osElapsed)/float64(osFrequency))

		fmt.Printf(" CPU timer: %v -> %v = %v\n", cpuStart, cpuEnd, cpuElapsed)
		fmt.Printf("  CPU freq: %v (estimated)\n", cpuFrequency)
	}

	return cpuFrequency
}

func main() {
	verbose = true
	var millisecondsToWait int64 = 10

	GetCPUTimerFreq(millisecondsToWait)
}
