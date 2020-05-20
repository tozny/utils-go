package utils

import (
	"time"
)

// Ready is a type of function that reports
// readiness of some state or action, returning
// bool for readiness and error (if any).
type Ready func() bool

// Await waits until the ready function is ready
// or errors, returning success and error (if any).
// To stop waiting, send on the stop channel.z
// It checks if the function is ready once and then retries
// the specified number of times with an exponential backoff between each attempt
func Await(ready Ready, maxRetries int) bool {
	for tries := 0; tries <= maxRetries; tries++ {
		success := ready()
		if !success {
			if tries != maxRetries {
				// exponentially back off before the next attempt
				// https://github.com/adonovan/gopl.io/blob/77e9f810f3c2502e9c641b97e09f9721424090f5/ch5/wait/wait.go#L30
				time.Sleep((1 * time.Second) << uint(tries))
			}
			continue
		}
		return true
	}
	return false
}

// AwaitInterval waits until the ready function is ready or errors, returning
// success and error (if any). It checks if the function is ready once, then
// waits the specified time interval (in seconds) and retries. If the specified
// timeout is past (taken in seconds) it will return false.
func AwaitInterval(ready Ready, interval int, timeout int) bool {
	timeoutTime := time.Now().Add(time.Duration(timeout) * time.Second)
	intervalTime := time.Duration(interval) * time.Second
	for timeoutTime.After(time.Now()) {
		success := ready()
		if !success {
			time.Sleep(intervalTime)
			continue
		}
		return true
	}
	return false
}
