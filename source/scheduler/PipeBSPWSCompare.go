package scheduler

import (
	"fmt"
	ws "proj3/WorkStealing"
	"proj3/utils"
	"time"
	c "proj3/constants"
)

// THIS VERSION IS FOR TESTING PURPOSES ONLY.

// This is a version of `PipeBSPWS` with the work stealing feature deactivated.
// It gives a way of visualizing the effects of work stealing when the overhead of creating
// workers and their DEQueues is factored out.
// Nonetheless, comparison with the `PipeBSP` is more accurate, since it
// it also takes into consideration the overhead of work stealing.

//=====================================================================================================================
// Pipeline phases callers
//=====================================================================================================================
// Run the phase 1 of the pipeline.
func RunP1(input <-chan ws.Runnable, worker *PipeWorker) {
	// retrieve tasks from 1st stage of pipeline assigned to `worker` and add them to it's DEqueue
	for i := 0; i < worker.numTasks; i++ {
		task := <- input
		worker.worker.AddTask(task)
	}
	// start execution/stealing
	worker.worker.RunNoWs(worker.done)
}

// Run the phase 1 of the pipeline.
func RunP2(input <-chan ws.Runnable, worker *PipeWorker) {
	for i := 0; i < worker.numTasks; i++ {
	// retrieve tasks from 2nd stage of pipeline assigned to `worker` and add them to it's DEqueue
		task := <- input
		worker.worker.AddTask(task)
	}
	// start execution/stealing
	worker.worker.RunNoWs(worker.done)
}

// Run the phase 3 of the pipeline.
func RunP3(input <-chan ws.Runnable, worker *PipeWorker) {
	for i := 0; i < worker.numTasks; i++ {
		// retrieve tasks from 3rd stage of pipeline assigned to `worker` and add them to it's DEqueue
		task := <- input
		worker.worker.AddTask(task)
	}
	worker.worker.RunNoWs(worker.done)
}

//==============================================================================
// Pipeline BSP with work stealing refinement execution
//==============================================================================
func RunPipeBSPWSCompare(config Config){
	//start timer
	startTime := time.Now()

	//--------------------------------------------------------------------------
	// Initialization
	//--------------------------------------------------------------------------
	
	// create a list of tasks based off of the data directories
	tasks := utils.CreateTasks(config.DataDirs)

	// compute number of threads to use in work stealing
	nThreads := config.ThreadCount
	if nThreads > len(tasks.Tasks){
		nThreads = len(tasks.Tasks)
	}

	// nSubThreads := config.SubThreadCount

	// timers for parallel section
	var totalParallelTime time.Duration
	startParallel := time.Now()

	//--------------------------------------------------------------------------
	// Execute pipeline
	//--------------------------------------------------------------------------
	
	// potentially process chunks of tasks to reduce memory usage

	// create chunks of tasks to process based on user input
	// if no input, defaults to all tasks
	var chunks []int
	if config.ChunkSize > 0{
		chunks = ChunksOfTasks(len(tasks.Tasks), config.ChunkSize)
	} else {
		chunks = []int{0, len(tasks.Tasks)}
	}

	// run the whole pipeline for each chunk of tasks
	for i := 0; i < len(chunks)-1; i++ {
		start := chunks[i]
		end := chunks[i+1]
		taskSubset := tasks.Tasks[start:end]

		// create a PipeContext for the pipeline
		pipeCtx := NewPipeContext(&config, c.PipePhases, len(taskSubset))
		
		// create groups of pipe workers for each phase and divide tasks among them
		// eg: if numThreads = 4, will create 4 PipeWorkers for each phase with 1/4 of the tasks each.
		pipeWorkers := make([][]*PipeWorker, c.PipePhases)
		for i := range pipeWorkers {
			pipeWorkers[i] = PrepareWorkers(nThreads, len(taskSubset))
		}

		// Start routines for each phase, each listening on the output channel of the previous phase
		for i := 0; i < nThreads; i++ {
			go RunPhase1(pipeCtx.channels[0], pipeWorkers[0][i])
			go RunPhase2(pipeCtx.channels[1], pipeWorkers[1][i])
			go RunPhase3(pipeCtx.channels[2], pipeWorkers[2][i])
	  	}
		// Send Phase1 tasks over the channel
		for i := range taskSubset {
			pipeCtx.channels[0] <- NewTaskPhase1(pipeCtx, &taskSubset[i], 0)
		}
		// close channel to signal end of tasks
		close(pipeCtx.channels[0]) 


		// Loop: for all pipeline phases:
		// - Wait for all tasks of a pipeline stage to finish
		// - Close the respective channels when they are finished 
		// - Signal workers to stop execution/stealing when phase is finished
		// This prevents goroutine leaks and wait for the full pipeline execution
		for i, wg := range pipeCtx.wgs {
			wg.Wait()
			if i < len(pipeCtx.wgs)-1 {
				// Phase 1 finished -> close channel receiving Phase 2 tasks
				close(pipeCtx.channels[i+1])
				// Phase 1 finished -> signal workers to stop execution/stealing
				close(pipeWorkers[i][0].done)
			}
		}
	}
	
	//--------------------------------------------------------------------------
	// Save results
	//--------------------------------------------------------------------------
		
	// elapsed time for parallel section
	totalParallelTime = time.Since(startParallel)

	// total elapsed time
	elapsedTime := time.Since(startTime)

	// write times + settings into JSON format 
	// Obs: PipeBSPWS mode = "pipebspws_<nSubThreads><_chunkSize>"
	var chunkSizeStr string
	if config.ChunkSize == 0 {
		chunkSizeStr = ""
	} else {
		chunkSizeStr = fmt.Sprintf("_%d", config.ChunkSize)
	}

	writeStr := fmt.Sprintf("{\"mode\": \"%s_%d%s\", \"threads\": %d, \"timeElapsed\": %f, \"timeParallel\": %f , \"datadir\": \"%s\"}\n", 
				config.Mode, config.SubThreadCount, chunkSizeStr ,nThreads, elapsedTime.Seconds(), totalParallelTime.Seconds(), config.DataDirs)
	
	// write results to file
	utils.WriteToFile(resultsPath, writeStr)
	
}
