package scheduler

import (
	"proj3/utils"
	"proj3/png"
	"fmt"
	"time"
	"os"
)

// Process images specified by 'config' and 'effects.txt', sequentially applying effects to each image.
func RunSequential(config Config) {
	// start timer for total elapsed time
	startTime := time.Now()
	
	// create a queue of tasks given data directories CMD inputs and effects.txt file
	taskQueue := utils.CreateTasks(config.DataDirs)

	// load image each image and apply effects sequentially
	for i := 0; i < len(taskQueue.Tasks); i++ {
		// load the image
		
		img, err := png.Load(taskQueue.Tasks[i].InPath)

		if err != nil{
			fmt.Println("Error loading image: ", err)
			os.Exit(1)
		}

		// apply the effects sequentially
		kernels := png.CreateKernels(taskQueue.Tasks[i].Effects)
		for _, kernel := range kernels {
			img.ApplyEffect(kernel)
			// invert image buffer for application of next effect (see png.Image struct definition)
			img.Final = 1 - img.Final
		}

		// save output and go to next image
		img.Save(taskQueue.Tasks[i].OutPath)
	}

	// compute elapsed time
	elapsedTime := time.Since(startTime)

	// write result into JSON format 
	writeStr := fmt.Sprintf("{\"mode\": \"%s\", \"threads\": %d, \"timeElapsed\": %f, \"timeParallel\": %f , \"datadir\": \"%s\"}\n", 
								config.Mode , 1, elapsedTime.Seconds(), 0.0, config.DataDirs)
	// write times to results text file
	utils.WriteToFile(resultsPath, writeStr)
}

