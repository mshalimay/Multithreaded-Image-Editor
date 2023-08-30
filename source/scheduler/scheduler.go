package scheduler

type Config struct {
	DataDirs string //Represents the data directories to use to load the images.
	Mode     string // Represents which scheduler scheme to use
	ThreadCount int // Runs parallel version with the specified number of threads
	SubThreadCount int // Only for PipeBSP modes. Number of routines a worker can spawn for the processing of each image.
	ChunkSize int // Only for PipeBSP modes. Number of images to be processed at the same time. Defaults to all images provided.
}

// Little modification from original: results file common to all scheduling schemes
const resultsPath = "./benchmark/results.txt"

//Run the correct version based on the Mode field of the configuration value
func Schedule(config Config) {
	if config.Mode == "s" {
		RunSequential(config)

	} else if config.Mode == "parfiles" {
		RunParallelFiles(config)

	} else if config.Mode == "parslices" {
		RunParallelSlices(config)
	
	} else if config.Mode == "pipebsp" {
		RunPipeBSP(config)
	
	} else if config.Mode == "pipebspws" {
		RunPipeBSPWS(config)

	} else if config.Mode == "pipebspwscompare" {
		RunPipeBSPWSCompare(config)
			
	} else {
		panic("Invalid scheduling scheme given.")
	}
}
