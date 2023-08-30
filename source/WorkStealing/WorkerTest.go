package workstealing

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

// mode 0: add tasks
// mode 1: execute tasks


type WorkerTest struct {
	queues 	[]*UDEqueue
	id 	  	int
	Mode 	atomic.Int32
	tasks 	[]Runnable
}

func NewWorkerTest(id int, queues []*UDEqueue) *WorkerTest {
	worker := &WorkerTest{queues: queues, id: id,  tasks: nil}
	worker.Mode.Store(1)
	return worker
}

func (w *WorkerTest) Run() {
	// pop a task from it's own queue and execute it
	task := w.queues[w.id].popBottom()
	var victim int

	for{
		// run in execute tasks mode. 
		// spinning on an `atomic` is not very efficient, but just for testing.
		for w.Mode.Load() == 1 && task != nil {
			// execute the task
			task.Execute(w.id)
			task = nil
			if !w.queues[w.id].IsEmpty() {
				task = w.queues[w.id].popBottom()
			}
		}

		// if own queue is empty, steal tasks from other threads
		for w.Mode.Load() == 1 && task == nil {
			// select a random victim. Keep drawing until it is not itself
			victim = rand.Intn(len(w.queues))
			for victim == w.id {
				victim = rand.Intn(len(w.queues))
			}

			// if victim's queue is not empty, steal a task from it; otherwise, select another victim
			if !w.queues[victim].IsEmpty() {
				task = w.queues[victim].PopTop()
				if task != nil {
					fmt.Printf("Worker %d stole task %d from worker %d\n", w.id, task.GetTaskID(), victim)
				}
			}
		}
		
		// run in create tasks mode
		for w.Mode.Load() == 0 {
			print := 0
			if print == 0 {
				fmt.Printf("Worker %d is in create tasks mode\n", w.id)
				print = 1
			}
			
			// add tasks to own queue
			for _, task := range w.tasks {
				time.Sleep(800 * time.Millisecond)
				fmt.Printf("Worker %d added task %d to its own queue\n", w.id, task.GetTaskID())
				w.queues[w.id].pushBottom(task)
			}
			// reset tasks
			w.tasks = nil

			// switch to execute mode
			w.Mode.Store(1)
		}
	}
}


func (w *WorkerTest) AddTask(task Runnable) {
	w.queues[w.id].pushBottom(task)
}


func (w *WorkerTest) NewTasks(tasks []Runnable) {
	w.tasks = tasks
}