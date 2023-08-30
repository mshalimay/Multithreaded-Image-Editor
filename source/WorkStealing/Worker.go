package workstealing

import (
	"math/rand"
)

// OBS: This worker does not `push` elements to the queue because it was not
// necessary for my use implementation. For an example of how one could look
// like, see `WorkerTest.go`.

// `Worker` is a struct that represents a thread in the work stealing scheduler.
// Each `Worker` access it's own queue among the `queues` slice and steal tasks
// from other threads by randomly selecting a queue and trying to `popTop` a task from it.
type Worker struct {
	queues 		[]*UDEqueue   // queues of `Runnable`s (one for each worker)
	tasksAdd 	[]Runnable	  // tasks to be added to the queue
	id 	  		int			  // id of the worker
}

// NewWorker returns a new `Worker` with the given id and queues.
func NewWorker(id int, queues []*UDEqueue) *Worker {
	worker := &Worker{queues: queues, id: id,  tasksAdd: nil}
	return worker
}

// `Run` in loop executing tasks from it's own queue or by stealing tasks from other threads.
// Will run in loop until a `done` signal is received.
func (w *Worker) Run(done <- chan struct{}) {
	var victim int
	// initialize `task` by popping an element from it's own queue
	task := w.queues[w.id].popBottom()

	// Loop: execute tasks (own or stolen) until a `done` signal is received
	for{
		select{
		
		// If `done` signal is received, stop working/stealing and return
		case <- done:
			return
		
		// Execute owned/stolen tasks
		default:
			// pop a task from it's own queue and execute it. 
			// Keep popping until queue is empty.
			for task != nil {
				// execute the task
				task.Execute(w.id)
				task = nil
				if !w.queues[w.id].IsEmpty() {
					task = w.queues[w.id].popBottom()
				}
			}

			// if own queue is empty, steal tasks from other threads
			for task == nil {
				victim = w.SelectRandomVictim()
				// if victim's queue is not empty, steal a task; otherwise, go to next victim
				if !w.queues[victim].IsEmpty() {
					task = w.queues[victim].PopTop()
				}
			}
		}
	}
}


// SelectRandomVictim returns a random index representing another worker.
func (w *Worker) SelectRandomVictim() int{
	// select a random victim. Keep drawing until it is not itself
	victim := rand.Intn(len(w.queues))
	for victim == w.id {
		victim = rand.Intn(len(w.queues))
	}
	return victim
}

// AddTask adds a task to the worker's queue.
func (w *Worker) AddTask(task Runnable) {
	w.queues[w.id].pushBottom(task)
}


// for debugging
func (w *Worker) GetTask(index int) (Runnable, bool) {
	circArray := (*CircularArray)(w.queues[w.id].tasks)
	
	if index < 0 || index >= circArray.GetCapacity() {
		return nil, false
	}
	
	return circArray.tasks[index], true
}


//==============================================================================
// Deactivate work stealing: comparison purposes only
//==============================================================================


// `Run` in loop executing tasks from it's own queue or by stealing tasks from other threads.
// Will run in loop until a `done` signal is received.
func (w *Worker) RunNoWs(done <- chan struct{}) {
	// initialize `task` by popping an element from it's own queue
	task := w.queues[w.id].popBottom()
	// Loop: execute tasks (own) until a `done` signal is received or tasks are done
	for{
		select{
		
		// If `done` signal is received, stop working/stealing and return
		case <- done:
			return
		
		// Execute owned/stolen tasks
		default:
			// pop a task from it's own queue and execute it. 
			// Keep popping until queue is empty.
			for task != nil {
				// execute the task
				task.Execute(w.id)
				task = nil
				if !w.queues[w.id].IsEmpty() {
					task = w.queues[w.id].popBottom()
				}
			}

			// No work stealing
			if task == nil {
				return
			}
		}
	}
}