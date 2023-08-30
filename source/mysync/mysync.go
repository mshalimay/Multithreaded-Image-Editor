package mysync

import (
	"sync/atomic"
	"runtime"
	"bytes"
	"strconv"
)

//==============================================================================
// atomicBoolean struct and methods
//==============================================================================

// atomicBoolean struct represents a boolean that can be atomically set and read
// @value: int32 value of the boolean. 0 = false, >0 = true
type atomicBoolean struct{
	value uint32
} 

// intToBool converts an int32 to a bool
// obs: 0 = false, otherwise = true
func intToBool(value uint32) bool{
	return value != 0
}

// boolToInt converts a bool to an int32
func boolToInt(value bool) uint32{
	if value {
		return 1
	}
	return 0
}

// NewatomicBool creates a new atomicBoolean struct and returns a pointer to it
func NewatomicBool(value bool) atomicBoolean{
	return atomicBoolean{value:boolToInt(value)}
}

// Get returns the value of the atomicBoolean
func (aBool *atomicBoolean) GetAndSet(newVal bool) bool{
	oldVal := atomic.SwapUint32(&aBool.value, boolToInt(newVal))
	return intToBool(oldVal)
}

// Set sets the value of atomicBoolean. Obs: not thread safe
func (aBool *atomicBoolean) Set(newVal bool){
	aBool.value = boolToInt(newVal)
}

//==============================================================================
// TAS lock struct and methods
//==============================================================================

// TASLock struct represents a test and set lock
// @state: pointer to an atomicBoolean struct representing the lock state
// 0 = unlocked, >0 = locked
type TASLock struct{
	state *atomicBoolean
}

// Creates a new TASLock struct and returns a pointer to it
func NewTasLock() TASLock{
	state := NewatomicBool(false)
	return TASLock{state: &state}
}

// Lock locks the TASLock
func (lock *TASLock) Lock() {
	for lock.state.GetAndSet(true){
		runtime.Gosched()
	}
}

// Unlock unlocks the TASLock
func (lock *TASLock) Unlock() {
	lock.state.Set(false)	
}

// ExecuteOne executes 'function' just one time when called by a 'counter' group of threads
// @counter: pointer to a counter variable. Used to keep track of the number of threads that have called 'function'
// @tLock: pointer to a TASLock struct used to synchronize access to 'counter'
// @nThreads: number of threads that will call 'function'. Passed as a copy.
func ExecuteOne(counter *int, tLock *TASLock,  nThreads int, function func()) {
	// if only one thread, execute function
	if nThreads == 1 {
		function()
		return
	}
	// fist thread execute the function and increment counter; 
	// other threads increment counter until it reaches nThreads 
	// then last thread resets counter
	tLock.Lock()
	if *counter == 0 {
		function()
		*counter++
	} else if *counter + 1 == nThreads {
		*counter = 0
	} else {
		*counter++
	}
	tLock.Unlock()
}

//==============================================================================
// Methods for debugging
//==============================================================================

// GetGID returns the goroutine id of the caller
func GetGID() uint64 {
    b := make([]byte, 64)
    b = b[:runtime.Stack(b, false)]
    b = bytes.TrimPrefix(b, []byte("goroutine "))
    b = b[:bytes.IndexByte(b, ' ')]
    n, _ := strconv.ParseUint(string(b), 10, 64)
    return n
}
