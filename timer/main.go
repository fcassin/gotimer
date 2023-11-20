package timer

// #cgo CFLAGS: -g -Wall
// #include <stdlib.h>
// #include "timer.h"
import "C"
import (
	"fmt"
	"strings"
	"time"
)

const TOTAL_SECTION_NAME = "total"
const sectionNameMaxLength = 24

var verbose bool
var cpuFrequency int64
var timingBySection = make(map[string]*timing)
var timings = make([]*timing, 0, 10)

var totalTiming = timing{
	section: TOTAL_SECTION_NAME,
}

type timing struct {
	section string
	start   int64
	end     int64
	count   int64
	elapsed float64
}

func readOSTimer() int64 {
	return time.Now().UnixMicro()
}

func getOSTimerFreq() int64 {
	return 1000000
}

func ReadCPUTimer() int64 {
	cvalue := C.ReadCPUTimer()
	return int64(cvalue)
}

func getCPUTimerFreq(millisecondsToWait int64) int64 {
	osFrequency := getOSTimerFreq()
	if verbose {
		fmt.Printf("   OS Freq: %v (reported)\n", osFrequency)
	}

	cpuStart := ReadCPUTimer()
	osStart := readOSTimer()
	var osEnd, osElapsed int64
	osWaitTime := osFrequency * millisecondsToWait / 1000
	for osElapsed < osWaitTime {
		osEnd = readOSTimer()
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

func Start(section string) {
	// NOTE: Handle a hierarchy of timers
	if cpuFrequency == 0 {
		cpuFrequency = getCPUTimerFreq(50)
	}

	if len(section) > sectionNameMaxLength {
		section = section[:sectionNameMaxLength]
	}

	var sectionTiming = timing{
		section: section,
	}

	timings = append(timings, &sectionTiming)

	timingBySection[section] = &sectionTiming

	var current = ReadCPUTimer()
	if totalTiming.start == 0 {
		totalTiming.start = current
	}

	sectionTiming.start = current
}

func Stop(section string) {
	var end = ReadCPUTimer()

	if len(section) > sectionNameMaxLength {
		section = section[:sectionNameMaxLength]
	}

	var timing = timingBySection[section]
	timing.end = end
	timing.count = timing.end - timing.start
	timing.elapsed = float64(timing.count) / float64(cpuFrequency/1000)

	totalTiming.end = end
	totalTiming.count = totalTiming.end - totalTiming.start
	totalTiming.elapsed = float64(totalTiming.count) / float64(cpuFrequency/1000)
}

func Output() {
	// NOTE: Should the output be generated here?
	// Seems weird. It's handy, but maybe timer shouldn't print
	// directly and should return data to the calling code
	// Maybe code to be put in a test/an example

	fmt.Println()

	var padding = strings.Repeat(" ", sectionNameMaxLength-len(totalTiming.section))
	fmt.Printf("%s%s: %10.3fms (CPU freq: %d)\n", padding, totalTiming.section, totalTiming.elapsed, cpuFrequency)

	for _, timing := range timings {
		var percent = 100 * float64(timing.count) / float64(totalTiming.count)
		var padding = strings.Repeat(" ", sectionNameMaxLength-len(timing.section))

		fmt.Printf("%s%s: %10.3fms (%5.2f%%)\n", padding, timing.section, timing.elapsed, percent)
	}
}

func main() {
	verbose = true
	var millisecondsToWait int64 = 10

	getCPUTimerFreq(millisecondsToWait)
}
