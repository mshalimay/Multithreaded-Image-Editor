package workstealing

import (
	"sync/atomic"
	"unsafe"
)


// UDEqueue is an unbounded double-ended queue built with a `CircularArray` of `Tasks`.
// The queue is "owned" by a thread in the sense that only one thread can push tasks to the `bottom` of the queue.
// Thieves can access the `top` of the queue to steal tasks.
// The owner only pop elements from the `bottom` of the queue.
type UDEqueue struct {
	tasks 			   	unsafe.Pointer // `CircularArray` of `Task`s; unsafe.Pointer is used to allow atomic operations
	bottom 	   			int64  		   // Points to the entry after the bottomost element of the queue.
	top 	   			int64		   // Points to the topmost element of the queue. Always increases.
}

// Examples of states and operations: 
// Bottom = 8; top = 2; capacity = 16 ==> Entries 2:7 contains `Task`s.
// If owner wants to `popBottom` an element: update bottom to 7 and retrieve the task at index 7.
// If thief wants to `popTop` an element: update top to 3 and retrieve the task at index 2.

// Bottom = 8; top = 18; capacity = 16 ==> Entries 2:(20%16) contains `Task`s.
// If owner wants to `popBottom` an element: update bottom to 7 and retrieve the task at index 7.
// If thief wants to `popTop` an element: increment top to 19 and retrieve the task at index (18 % 16 = 2)

// Bottom = 8, top = 7, capacity = 16 ==> Only one `Task` in the queue; thives and owner compete for it.
// Bottom = 8, top = 8, capacity = 16 ==> Queue is empty


// NewUDEqueue returns a new UDEqueue
func NewUDEqueue(initialLogCapacity int) *UDEqueue {
	circArray := NewCircularArray(initialLogCapacity)
	return &UDEqueue{unsafe.Pointer(circArray), 0, 0}
}


// IsEmpty returns true if the queue is empty, false otherwise. Both thieves and the owner can call this method.
// Obs: Calling this method before trying to `pop` is advised, because it is less expensive.
func (u *UDEqueue) IsEmpty() bool {
	//NOTE: If this method is only called by thieves, do not need an atomic operation to load `top`.
	//	If the owner also calls it, needs.

	// NOTE: The order of reads matter. Since top always increase, load it first 
	// because if `bottom` <= `oldTop`, necessarily `bottom` <= any value for `top`.
	oldTop := atomic.LoadInt64(&u.top)
	
	return u.bottom <= oldTop
	// NOTE: Does not need a atomic to load `bottom` above; only consequence is 
	// more false positives (i.e., queue is not empty but thieves think it is). 
	// But notice that Go's race detector will throw a data race.
}

// PushBottom pushes a task to the bottom of the queue. Only the owner of the queue calls this method.
func (u *UDEqueue) pushBottom(task Runnable) {
	// Get current top of the queue
	oldTop := atomic.LoadInt64(&u.top)
	
	// Check if there is still space in the queue.
	size := u.bottom - oldTop
	tasks := (*CircularArray)(u.tasks)


	// if there is no space, resize the queue
	if (int(size) >= tasks.GetCapacity() -1) {
		// an atomic store needs to be used to communicate to all threads of the new queue
		atomic.StorePointer(&u.tasks, unsafe.Pointer(tasks.Resize(int(oldTop), int(u.bottom))))
	}
	// Obs: this might resize when there is still space, because thieves might have 
	// stolen tasks in between. Could change to a retry strategy if memory becomes a concern.


	// put the task in the queue
	(*CircularArray)(u.tasks).PutTask(int(u.bottom), task)
	// obs: dont need an atomic load for the bottom, since only the owner 
	// of the queue (the only one using `pushBottom`) will update the bottom.

	// update bottom pointer
	atomic.AddInt64(&u.bottom, 1)

	// REVIEW: see if need an atomic operation above. Only the owner modifies 
	// the bottom, but an atomic is used to make other threads aware of the
	//  new bottom. It might not be necessary though; consequence I think would 
	// be more false "emptys" when thieves try to steal.
}


// PopTop pops a task from the top of the queue. Only thieves call this method.
// Obs: This method might return nil even if the queue is not empty.
// This is not a problem; thieves will just try to steal again.
func (u *UDEqueue) PopTop() Runnable {
	
	// Get the index of the element to steal from the top part of the queue.
	oldTop := atomic.LoadInt64(&u.top)
	
	// If the queue is empty, return nil.
	if (u.bottom <= oldTop) {
		return nil
	}
	// NOTE: can use an atomic for `bottom`, above but not necessary; consequence is more `nil` returns. 
	// But notice that Go will throw a data race.

	// Not empty -> try to get a task. 
	task := (*CircularArray)(u.tasks).GetTask(int(oldTop))

	// CAS re-confirms the entry being pointed to is still the same. 
	// If `oldTop` is still the queue's top, then return the task.
	// Otherwise, someone else won the race to get a task =>  give up and try stealing again.
	if atomic.CompareAndSwapInt64(&u.top, oldTop, oldTop + 1) {
		return task
	}
	// Obs: Notice we do not have an ABA problem here because `top` always increases.
	// Eg: if oldTop = 5 and other thread steals first, it updates `top` to 6.
	// CAS returns false to the loser irrespective if other element was added to the
	// [5] entry by the owner. This would not be true if `top` could decrease.

	return nil
}

// PopBottom pops a task from the bottom of the queue. Only the owner calls this method.
func (u *UDEqueue) popBottom() Runnable {
	// Update the bottom of the queue.
	// Atomic is used here to communicate to all threads that the bottom of the queue was updated;
	// this is relevant in the case the queue becomes empty, so that a thief does not steal from an empty queue
	atomic.AddInt64(&u.bottom, -1)

	// Get the top of the queue at this snapshot; this will be used to resolve conflits 
	// in case case the task is the last element and there are thieves trying to get it too.
	oldTop := atomic.LoadInt64(&u.top)
	
	// if size < 0, the queue is empty but was not cleaned up. 
	// => Reset the queue (i.e. `top` and `bottom` point to the same element) and return nil.
	size := int(u.bottom - oldTop)
	if (size < 0){
		atomic.SwapInt64(&u.bottom, oldTop)
		return nil
	}

	// not empty -> try to get a task.
	task := (*CircularArray)(u.tasks).GetTask(int(u.bottom))

	// if distance between top and bottom is large, no conflicts, just return task.
	// eg: if bottom = 8, top = 2, capacity = 16 => Entries 2:7 contains `Task`s. 
	// Thieves will be stealing from 7 and owner from 2, so no conflicts.
	if (size > 0) {
		return task
	}

	// If size == 0, owner of the queue and thieves competing for the last element.
	// CAS operator will resolve the conflict giving the task to the fastest thread.
	// If someone else got the task, the owner resets the queue to empty and return nil.
	if !atomic.CompareAndSwapInt64(&u.top, oldTop, oldTop + 1) {
		// task to return is nil
		task = nil
		
		// Reset the queue
		// Obs:oldTop + 1 -> bottom because if a thief won the race, it will have
		// incremented the top, to reset the queue needs to increment the bottom.
		// eg: bottom = 8, top = 7; thief wins => newTop = 8; reset making oldTop + 1 = 7 + 1 = new top = 8
		atomic.SwapInt64(&u.bottom, oldTop + 1)

		// REVIEW: I believe an atomic is needed above, so that other thieves know the 
		// queue was reset. But I'm not sure. It is possible it is not needed because 
		//at this point it is known the queue is empty (bottom <= top) in the branches 
		// before, so thieves will not try to steal from it anyway.
	}
	return task
}

func (u *UDEqueue) GetCapacity() int {
	return (*CircularArray)(u.tasks).GetCapacity()
}