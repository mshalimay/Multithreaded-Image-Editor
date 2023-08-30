package main

import (
	"fmt"
	"math/rand"
	"os"
	ws "proj3/WorkStealing"
	"sync"
	"time"
)

const busyWorker = 2

type SleepTask struct {
	wID 		int
	wg 			*sync.WaitGroup
	taskID 		int
}

func NewSleepTask(workerID int, taskID int, wg *sync.WaitGroup) *SleepTask {
	return &SleepTask{wID: workerID, taskID:taskID, wg: wg}
}  

func (st *SleepTask) Execute(wId int) {
    
	var sleepFor int
	if st.wID > busyWorker {
		sleepFor = 2 + rand.Intn(2)
	
	// worker 2 is the busy worker
	} else if st.wID == busyWorker {
		sleepFor = 1 + rand.Intn(3)
		// sleepFor = 1 + rand.Intn(1)
	} else {
		sleepFor = rand.Intn(2)
	} 

	if wId != st.wID {
		fmt.Printf("Worker %d exec task %d on behalf of worker %d\n", wId, st.taskID, st.wID)
	} else {
		fmt.Printf("Worker %d executing task %d\n", wId, st.taskID)
	}
    fmt.Printf("Worker %d sleeping for %d seconds\n", wId, sleepFor)
    time.Sleep(time.Duration(sleepFor) * time.Second)
	st.wg.Done()
}

func (st *SleepTask) GetTaskID() int {
	return st.taskID
}


func main() {

	file, err := os.Create("log.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Redirect standard output to log file
	os.Stdout = file


	// initial paramaters
	numWorkers := 8
	logCapacity := 8

	// slice of workers and queues
	workers := make([]*ws.WorkerTest, numWorkers)
	queues := make([]*ws.UDEqueue, numWorkers)

	// Initialize the workers and their queues.
	for i := range workers {
		queues[i] = ws.NewUDEqueue(logCapacity)
		workers[i] = ws.NewWorkerTest(i, queues)
	}

	// Generate tasks and add them to the workers' queues.
	var wg sync.WaitGroup

	taskID := 0
	for i := range workers {
		// random number of tasks per queue
	 	tasksPerQueue:= 0
		if i == busyWorker {
			tasksPerQueue = 20 + rand.Intn(10)
		}else{ 
			tasksPerQueue = 0
		}
		
		// fmt.Printf("Worker %d will have %d tasks\n", i, tasksPerQueue)
		for j := 0; j < tasksPerQueue; j++ {
			wg.Add(1)
			taskID++
			workers[i].AddTask(NewSleepTask(i, taskID, &wg))
		}
	}
	

	// Start the workers.
	for _, worker := range workers {
		go func(w *ws.WorkerTest) {
			w.Run()
		}(worker)
	}

	// create more work for the busyWorker
	tasksPerQueue := 10 + rand.Intn(40)
	newTasks := make([]ws.Runnable, 0)

	for j := 0; j < tasksPerQueue; j++ {
		wg.Add(1)
		taskID++
		newTasks = append(newTasks, NewSleepTask(busyWorker, taskID, &wg))
	}


	workers[busyWorker].NewTasks(newTasks)
	

	workers[busyWorker].Mode.Store(0)
	fmt.Printf("Switched worker %d to create tasks mode\n", busyWorker)

	// Wait for all workers to finish.
	wg.Wait()
	fmt.Printf("Total tasks: %d\n", taskID-1)

}