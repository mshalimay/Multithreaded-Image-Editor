package main

import (
	"fmt"
	"os"
	"proj3/scheduler"
	"strconv"
	"time"
)

const usage = "Usage: editor data_dir mode [number of threads]\n" +
	"data_dir = The data directory to use to load the images.\n" +
	"mode     = (s) run sequentially, (parfiles) process multiple files in parallel, (parslices) process slices of each image in parallel" +
				"(pipebsp) run the pipeline version of the program, (pipebspws) run the pipeline version of the program with work stealing.\n" +
	"[number of threads] = Runs the parallel version of the program with the specified number of threads." +
	"[number of sub-threads] = Only for PipeBSP modes. Number of sub-routines each thread can spawn for image processing in slices. Defaults to 1."+
	"[Chunk size] = Only for PipeBSP modes. Number of images to be processed at the same time. Defaults to all images provided.\n]"


func main() {

	// for debugging

	if len(os.Args) < 2 {
		fmt.Println(usage)
		return
	}

	config := scheduler.Config{DataDirs: "", Mode: "", ThreadCount: 0, SubThreadCount: 0}
	config.DataDirs = os.Args[1]

	// Parse command line arguments
	
	// If # threads not specified, default to sequential mode
	if len(os.Args) > 3 {
		config.Mode = os.Args[2]
		threads, _ := strconv.Atoi(os.Args[3])
		config.ThreadCount = threads
	} else {
		config.Mode = "s"
	}

	// If # sub-threads not specified, default to 1
	if len(os.Args) > 4 {
		subThreads, _ := strconv.Atoi(os.Args[4])
		config.SubThreadCount = subThreads
	} else {
		config.SubThreadCount = 1
	}

	if len(os.Args) > 5 {
		chunkSize, _ := strconv.Atoi(os.Args[5])
		config.ChunkSize = chunkSize
	} else {
		config.ChunkSize = 0
	}

	start := time.Now()
	scheduler.Schedule(config)
	end := time.Since(start).Seconds()
	fmt.Printf("%.2f\n", end)

}
