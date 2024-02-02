package timer

// #cgo CFLAGS: -g -Wall
// #include <stdlib.h>
// #include "timer.h"
import "C"
import (
	"fmt"
	"os"
	"time"
)

const TIMER_ENV_VAR = "TIMER"

const TOTAL_ANCHOR_NAME = "total"
const anchorNameMaxLength = 18

const maxHandledAnchors = 1000

var verbose bool
var cpuFrequency int64

var index int
var anchors []*anchor = make([]*anchor, maxHandledAnchors)
var anchorsByName = make(map[string]*anchor, maxHandledAnchors)

var totalTiming *timing = &timing{}
var currentAnchor *anchor
var currentTiming *timing

var totalAnchor = &anchor{
	name: TOTAL_ANCHOR_NAME,
}

type timing struct {
	start int64
	// Do we need to note the stop time here?

	previous *timing
	anchor   *anchor
}

type anchor struct {
	hits    int64
	depth   int64
	tscount int64
	bytes   int64
	elapsed float64

	name string

	active bool

	parent *anchor
	latest *timing
}

func readOSTimer() int64 {
	return time.Now().UnixMicro()
}

func getOSTimerFreq() int64 {
	return 1000000
}

func readCPUTimer() int64 {
	cvalue := C.ReadCPUTimer()
	return int64(cvalue)
}

func getCPUTimerFreq(millisecondsToWait int64) int64 {
	osFrequency := getOSTimerFreq()
	if verbose {
		fmt.Printf("   OS Freq: %v (reported)\n", osFrequency)
	}

	cpuStart := readCPUTimer()
	osStart := readOSTimer()
	var osEnd, osElapsed int64
	osWaitTime := osFrequency * millisecondsToWait / 1000
	for osElapsed < osWaitTime {
		osEnd = readOSTimer()
		osElapsed = osEnd - osStart
	}

	cpuEnd := readCPUTimer()
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

// NOTE: Do we need an init function?
// Reset fullfills a similar role, might simply rename it?
func Reset() {
	index = 0
	anchors = make([]*anchor, maxHandledAnchors)
	anchorsByName = make(map[string]*anchor, maxHandledAnchors)

	totalTiming = &timing{}
	currentAnchor = nil
	currentTiming = nil

	totalAnchor = &anchor{
		name: TOTAL_ANCHOR_NAME,
	}
}

/*
Start begins recording time for the specified anchor name.
Stop MUST be called with the same anchor name at some point. Deferring the Stop
call might be a good idea to time a complete block.

Profiler can be disabled by setting TIMER env variable to "0".
*/
func Start(anchorName string) {
	StartThroughput(anchorName, 0)
}

func StartThroughput(anchorName string, processedBytes int64) {
	if os.Getenv(TIMER_ENV_VAR) == "0" {
		return
	}

	if cpuFrequency == 0 {
		cpuFrequency = getCPUTimerFreq(50)
	}

	if len(anchorName) > anchorNameMaxLength {
		anchorName = anchorName[:anchorNameMaxLength]
	}

	var startingAnchor *anchor
	var exists bool

	startingAnchor, exists = anchorsByName[anchorName]
	if !exists {
		startingAnchor = &anchor{
			name:   anchorName,
			active: true,
		}

		anchorsByName[anchorName] = startingAnchor
		index = index + 1
		anchors[index] = startingAnchor

		if currentAnchor != nil {
			startingAnchor.depth = currentAnchor.depth + 1
		}

		startingAnchor.parent = currentAnchor
	}

	// NOTE: Need to keep track of the previous anchor as well?
	startingAnchor.hits = startingAnchor.hits + 1
	startingAnchor.bytes = startingAnchor.bytes + processedBytes
	currentAnchor = startingAnchor

	// Clock reading, limit operations as much as possible from now on
	var current = readCPUTimer()

	var startingTiming *timing
	// TODO: Create a large pool of objects and reuse them instead of creating
	// and discarding them regularly? Interesting thing to look at
	startingTiming = &timing{
		start:    current,
		previous: currentTiming,
		anchor:   startingAnchor,
	}

	startingAnchor.latest = startingTiming

	if totalTiming.start == 0 {
		totalTiming.start = current
		totalTiming.anchor = totalAnchor
		totalAnchor.latest = totalTiming
	}

	if currentTiming != nil {
		currentTiming.anchor.active = false
		currentTiming.anchor.tscount = currentTiming.anchor.tscount + current - currentTiming.start
	}

	currentTiming = startingTiming
}

/*
Stop ends the recording for the specified anchor name.
*/
func Stop(anchorName string) {
	if os.Getenv(TIMER_ENV_VAR) == "0" {
		return
	}

	var end = readCPUTimer()

	if len(anchorName) > anchorNameMaxLength {
		anchorName = anchorName[:anchorNameMaxLength]
	}

	var anchor = anchorsByName[anchorName]

	// Note: Anchor is about hierarchy
	// Note: Timing is about recursion

	var previousTiming *timing = anchor.latest.previous
	if previousTiming != nil {
		previousTiming.start = end
		previousTiming.anchor.active = true
	}

	if anchor.parent != nil {
		anchor.parent.latest.start = end
		anchor.parent.active = true
	}

	currentAnchor = anchor.parent
	currentTiming = previousTiming

	anchor.tscount = anchor.tscount + end - anchor.latest.start
	anchor.elapsed = float64(anchor.tscount) / float64(cpuFrequency/1000)

	totalAnchor.tscount = end - totalTiming.start
	totalAnchor.elapsed = float64(totalAnchor.tscount) / float64(cpuFrequency/1000)
}

/*
Output displays computed information for the current timer execution, to the
standard output.
*/
func Output() {
	if os.Getenv(TIMER_ENV_VAR) == "0" {
		return
	}

	// NOTE: Should the output be generated here?
	// Seems weird. It's handy, but maybe timer shouldn't print
	// directly and should return data to the calling code
	// Maybe code to be put in a test/an example

	fmt.Println()

	var padding = anchorNameMaxLength
	fmt.Printf("%*s: %10.3fms (CPU freq: %d)\n", padding, totalAnchor.name,
		totalAnchor.elapsed, cpuFrequency)

	for index, anchor := range anchors {
		if index == 0 {
			// Skip the first timing section for now
			continue
		}

		if anchor == nil {
			break
		}

		var percent = 100 * float64(anchor.tscount) / float64(totalAnchor.tscount)
		var padding = anchorNameMaxLength + 2*anchor.depth

		if anchor.bytes == 0 {
			fmt.Printf("%*s: %10.3fms (%5.2f%%) -- calls: %d\n", padding, anchor.name,
				anchor.elapsed, percent, anchor.hits)
		} else {
			var megabytes = float64(anchor.bytes) / (1024 * 1024)
			var gigabytes = float64(anchor.bytes) / (1024 * 1024 * 1024)
			var throughput = gigabytes / (float64(anchor.tscount) / float64(cpuFrequency))

			fmt.Printf("%*s: %10.3fms (%5.2f%%) -- calls: %d, %7.2fMB at %5.3fGB/s\n",
				padding, anchor.name, anchor.elapsed, percent, anchor.hits, megabytes, throughput)
		}
	}
}
