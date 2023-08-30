package scheduler

import (
	"proj3/png"
	"proj3/utils"
	"sync"
	"fmt"
	"time"
)

// Pick tasks from 'taskQueue' and apply effects to the images represented by them.
func ExecuteTask(taskQueue *utils.TaskQueue, wg *sync.WaitGroup){
	// pick a task from the queue thread-safely
	task := taskQueue.Dequeue()

	// loop: while there are tasks to be done, pick from queue and apply effects to image
	for task != nil {
		// load image and apply effects
		img, _ := png.Load(task.InPath)
		
		// create a slice of kernels representing each effect
		kernels := png.CreateKernels(task.Effects)

		// apply the effects to the image in sequence
		for _, kernel := range kernels {
			img.ApplyEffect(kernel)
			// invert image buffer for application of next effect (see png.Image struct definition)
			img.Final = 1 - img.Final
		}

		// save output and go to next image
		img.Save(task.OutPath)
		task = taskQueue.Dequeue()
	}
	// signal that this thread is done
	wg.Done()
}


// Process images specified by 'config' and 'effects.txt' deploying 'config.ThreadCount' 
// goroutines to apply effects to each image in parallel. 
func RunParallelFiles(config Config) {
	// start timer for total elapsed time
	startTime := time.Now()

	// create a queue of tasks given data directories CMD inputs and effects.txt file
	taskQueue := utils.CreateTasks(config.DataDirs)

	// compute number of threads to use; if more threads than tasks, use number of tasks
	nThreads := config.ThreadCount
	if nThreads > len(taskQueue.Tasks){
		nThreads = len(taskQueue.Tasks)
	}

	// wait group to wait until all threads are done
	var wg sync.WaitGroup
	
	// start timer for parallel tasks
	parallelTime := time.Now()
	// deploy go routines to apply effects to each image
	for i:=0; i < nThreads; i++{
		wg.Add(1)
		go ExecuteTask(taskQueue, &wg)
	}
	// wait for all threads to finish
	wg.Wait()
	
	// compute elapsed time for parallel section
	totalParallelTime := time.Since(parallelTime)

	// compute total elapsed time
	elapsedTime := time.Since(startTime)

	// write result into JSON format 
	writeStr := fmt.Sprintf("{\"mode\": \"%s\", \"threads\": %d, \"timeElapsed\": %f, \"timeParallel\": %f , \"datadir\": \"%s\"}\n", 
								config.Mode ,nThreads, elapsedTime.Seconds(), totalParallelTime.Seconds(), config.DataDirs)
	// write elapsed time to a text file
	utils.WriteToFile(resultsPath, writeStr)
}


