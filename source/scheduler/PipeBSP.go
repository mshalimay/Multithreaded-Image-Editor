package scheduler

import (
	"fmt"
	"proj3/utils"
	"time"
	ws "proj3/WorkStealing"
	c "proj3/constants"
)

//=====================================================================================================================
// Image processing using pipeline and BSP model.
// The strucutre is similar to `PipeBSPWS` but without the work stealing refinement.
// In particular, `Worker`s holds a slice (not a DEQuue) of `ws.Runnable` tasks
// for better comparison to the the work stealing version. See `PipeBSPWS` for the struct definitions.

// - Pipeline: load image -> apply effects -> save image
// - For each image in each phase, `Task`s implementing the `ws.Runnable` interface
// 	 are created and distributed among workers. Workers execute these tasks in parallel
//   and might steal tasks from one another if their own DEqueue is empty.

// - In phase2, each worker may spawn sub-threads to process slices of the image in parallel.
//   The synchronization of the sub-threads is done using a barrier from one effect to the next.
//=====================================================================================================================


//=====================================================================================================================
// Pipeline phases callers
//=====================================================================================================================

// Phase 1: load images and build kernels
func Run1(input <-chan ws.Runnable) {
	
	// iterate over phase 1 tasks received from previous phase and execute
	for task := range input {
	  task.Execute(0)
	}
}

// Phase 2: Apply effects
func Run2(input <-chan ws.Runnable) {
	// iterate over phase 2 tasks received from previous phase and execute
	for task := range input {
	  task.Execute(0)
	}
}

// Phase 3: Save new images
func Run3(input <-chan ws.Runnable){
	// iterate over phase 3 tasks received from previous phase and execute
	for task := range input {
	  task.Execute(0)
	}	
}

//==============================================================================
// Pipeline BSP execution
//==============================================================================
func RunPipeBSP(config Config){

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

		// Start workers for each phase, each listening on the output channel of the previous phase
		for i := 0; i < nThreads; i++ {
		  	go Run1(pipeCtx.channels[0])
		  	go Run2(pipeCtx.channels[1])
		  	go Run3(pipeCtx.channels[2])
		}

		// Create Tasks Phase 1 and send them over the pipeline
		for i := range taskSubset {
			pipeCtx.channels[0] <- NewTaskPhase1(pipeCtx, &taskSubset[i], 0)
		}
		// close channel to signal end of tasks
		close(pipeCtx.channels[0]) 

		// Loop: for all pipeline phases:
		// - Wait for all tasks of a pipeline stage to finish
		// - Close the respective channels when they are finished 
		// This prevents goroutine leaks and wait for the full pipeline execution
		for i, wg := range pipeCtx.wgs {
			wg.Wait()
			if i < len(pipeCtx.wgs)-1 {
				// Phase 1 finished -> close channel receiving Phase 2 tasks
				// Phase 2 finished -> close channel receiving Phase 3 tasks
				close(pipeCtx.channels[i+1])
			}
		}
	}
	
	//=============================================================================
	// Save results
	//=============================================================================

	// elapsed time for parallel section
	totalParallelTime = time.Since(startParallel)

	// total elapsed time
	elapsedTime := time.Since(startTime)

	// write times + settings into JSON format 
	// Obs: PipeBSP mode = "pipebspws_<nSubThreads><_chunkSize>"
	
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
