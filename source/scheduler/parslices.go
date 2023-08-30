package scheduler

import (
	"sync"
	"proj3/png"
	"proj3/utils"
	"fmt"
	"time"
	"math"
)


// 'ImageSlice' contains indexes representing a slice of an image
type ImageSlice struct {
	XStart int
	XEnd   int
	YStart int
	YEnd   int
}

// Divide an image into 'numSlices' slices by row.
// Returns a slice of 'ImageSlice' structs containg indexes for each slice.
// @img: pointer to the image to be divided
// @numSlices: number of slices to divide the image into
func SlicesByRow(img *png.Image, numSlices int) []ImageSlice{
	// compute number of rows per slice
	nRows := img.Bounds.Dy()
	rowsPerSlice := int(math.Ceil(float64(nRows) / float64(numSlices)))
	
	// slice of 'ImageSlice' structs to be filled with indexes for each slice
	slices := make([]ImageSlice, numSlices)
	
	// loop: compute indexes for each slice
	for i := 0; i < numSlices; i++ {
		// compute start row index
		slices[i].YStart = i * rowsPerSlice
		
		// truncate start row index if exceeds image bounds
		if slices[i].YStart > nRows {
			slices[i].YStart = nRows
		}

		// compute end row index
		slices[i].YEnd = slices[i].YStart + rowsPerSlice
		// truncate end row index if exceeds image bounds
		// obs: this will cause last slice to pick up the remaining rows
		if slices[i].YEnd > nRows {
			slices[i].YEnd = nRows
		}
	
		// set x indexes to full image width
		slices[i].XStart = 0
		slices[i].XEnd = img.Bounds.Dx()
	}
	return slices
}

// Process images specified by 'config' and 'effects.txt' dividing them into slices 
// and deploying 'config.ThreadCount' goroutines to apply effects to each slice. 
// Obs: Each image is loaded, processed and saved at a time.
func RunParallelSlices(config Config) {
	//start timer
	startTime := time.Now()

	// create a queue of tasks given data directories CMD inputs and effects.txt file
	taskQueue := utils.CreateTasks(config.DataDirs)
	
	// compute number of threads to use
	nThreads := config.ThreadCount
	if nThreads > len(taskQueue.Tasks){
		nThreads = len(taskQueue.Tasks)
	}

	var wgEffect sync.WaitGroup
	// cumulative time of all parallel tasks
	var totalParallelTime time.Duration

	// loop: load each image from the queue, separate into slices, deploy go routines to apply effects to each slice
	for i := 0; i < len(taskQueue.Tasks); i++ {
		// load the image
		img, _ := png.Load(taskQueue.Tasks[i].InPath)
		
		// create image slices
		slices := SlicesByRow(img, nThreads)
		
		// create a sice of kernels representing each effect to be acccessed by all threads
		kernels := png.CreateKernels(taskQueue.Tasks[i].Effects)

		// start timer for parallel section
		startParallel := time.Now()

		// deploy go routines to apply effects to each slice
		for _, kernel := range kernels {
			for j := 0; j < nThreads; j++ {
				wgEffect.Add(1)
				go img.ApplyEffectSlice(kernel, slices[j].YStart, slices[j].YEnd, slices[j].XStart, slices[j].XEnd, &wgEffect)
			}
			// wait for all effects to be applied before applying next effect
			wgEffect.Wait()
			// invert image buffer to apply next effect (see Image definition in png.go)
			img.Final = 1 - img.Final
		}
		// compute elapsed time for parallel section and accumulate
		totalParallelTime += time.Since(startParallel)
		
		// save processed image
		img.Save(taskQueue.Tasks[i].OutPath)
	}
	// compute total elapsed time
	elapsedTime := time.Since(startTime)

	// write result into JSON format 
	writeStr := fmt.Sprintf("{\"mode\": \"%s\", \"threads\": %d, \"timeElapsed\": %f, \"timeParallel\": %f , \"datadir\": \"%s\"}\n", 
								config.Mode ,nThreads, elapsedTime.Seconds(), totalParallelTime.Seconds(), config.DataDirs)
	// write elapsed time to a text file
	utils.WriteToFile(resultsPath, writeStr)

}
