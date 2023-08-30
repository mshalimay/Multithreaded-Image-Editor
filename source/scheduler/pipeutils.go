// Group of methods and structs for pipeline execution

package scheduler

import (
	ws "proj3/WorkStealing"
	"proj3/constants"
	"proj3/png"
	"proj3/utils"
	"sync"
)

// syncContext contains elements to synchronize sub-threads during image processing.
type syncContext struct{
	mutex 		*sync.Mutex
	cond  		*sync.Cond
	wg 			*sync.WaitGroup
	counter 	int
	nThreads 	int
}
func NewSyncContext(nThreads int) *syncContext{
	var mutex sync.Mutex
	cond := sync.NewCond(&mutex)
	return &syncContext{mutex: &mutex, cond: cond, wg: &sync.WaitGroup{}, counter: 0,  nThreads: nThreads}
}

// PipeContext contains parameters of the overall pipeline
// Obs: this useful because the `Tasks` themselves create the next `Tasks` to be executed.
// Thus, they need to know the parameters to create the new tasks, the channels to send the next `Task` to, etc
type PipeContext struct {
	config 		*Config					// contains parameters as numThreads, numSubThreads, etc
	channels	[]chan ws.Runnable		// all channels of the pipeline
	wgs 		[]*sync.WaitGroup		// wait groups of each pipeline phase to signalize when all tasks are done
}

// Create a new PipeContext with `nPhases` channels and WaitGroups and `nTasks` tasks per channel.
func NewPipeContext(config *Config, nPhases int, nTasks int) *PipeContext{
	channels := make([]chan ws.Runnable, nPhases)
	wgs := make([]*sync.WaitGroup, nPhases)
	for i := range channels {
		channels[i] = make(chan ws.Runnable, nTasks)
		wg := &sync.WaitGroup{}
		wg.Add(nTasks)
		wgs[i] = wg
	}
	return &PipeContext{config: config, channels: channels, wgs: wgs}
}

// `InitTaskStealing` creates a slice of `nWorkers` workers and DEQues to hold `Task`s for execution.
// @memo: `worker` represents a thread executing tasks; a worker holds it's own queue
// of tasks to execute and might steal from other workers when it's own queue is empty.
func InitTaskStealing(nWorkers int) []*ws.Worker{
	workers := make([]*ws.Worker, nWorkers)
	dequeues := make([]*ws.UDEqueue, nWorkers)

	// Create DEQueues to hold tasks for each worker
	for i := range workers {
		dequeues[i] = ws.NewUDEqueue(constants.InitLogCapacity)	
	}

	// Create workers; workers have access to all DEQueues (for stealing)
	for i := range workers {
		workers[i] = ws.NewWorker(i, dequeues)
	}
	return workers
}

// Divide a group of `tasks` for the full pipeline into Chunks of size `chunkSize`.
// Example: if 1000 images and chunkSize = 100, returns [0, 100, 200, ..., 1000]
func ChunksOfTasks(numTasks, chunkSize int) []int {
	nChunks := (numTasks + chunkSize - 1) / chunkSize

	indexes := make([]int, nChunks+1)

	indexes[0] = 0
	for i := 1; i <= nChunks; i++ {
		if i == nChunks {
			indexes[i] = numTasks
		} else {
			indexes[i] = i * chunkSize
		}
	}
	return indexes
}

//=============================================================================
// Phase 1: Load images and build kernels
//=============================================================================

// TaskPhase1 implements `ws.Runnable` interface.
// Each image to be loaded is associated to a `TaskPhase1`.
type TaskPhase1 struct{
	pipeCtx 	*PipeContext	// parameters of the overall pipeline
	baseTask 	*utils.Task		// struct containing info of the image to be loaded	
	curPhase 	int				// pipeline phase this task belongs to	
}

func NewTaskPhase1(pipeCtx *PipeContext, baseTask *utils.Task, curPhase int) *TaskPhase1{
	return &TaskPhase1{pipeCtx: pipeCtx, baseTask: baseTask, curPhase: curPhase}
}

// Loads the image from disk and build the `Kernel` for the effects to be applied.
func (t *TaskPhase1) Execute(wID int){
	// load image from disk
	img, _ := png.Load(t.baseTask.InPath)

	// create a kernel based on the effects to be applied to the image
	kernels := png.CreateKernels(t.baseTask.Effects)

	// create a task for phase of next pipeline stage and send over the respective channel
	taskPhase2 := NewTaskPhase2(t.pipeCtx, img, kernels, t.baseTask, t.curPhase+1)
	t.pipeCtx.channels[t.curPhase+1] <- taskPhase2

	// signalize this task is done to the go-routine managing the overall pipeline
	t.pipeCtx.wgs[t.curPhase].Done()
}

// Not used; just to implement the `ws.Runnable` interface.
func (t *TaskPhase1) GetTaskID() int{return 0}

//==============================================================================
// Phase 2: Image processing
//==============================================================================

// TaskPhase2 implements `ws.Runnable` interface.
// Each image to be processed is associated to a `TaskPhase2`.
type TaskPhase2 struct {
	pipeCtx 		*PipeContext		// parameters of the overall pipeline
	img 			*png.Image			// image to be processed
	kernels 		[]*png.Kernel		// effects to be applied to the image
	baseTask 		*utils.Task			// contains info of the image being processed	
	curPhase 		int					// pipeline phase this task belongs to	
}

func NewTaskPhase2(pipeCtx *PipeContext, img *png.Image, kernels []*png.Kernel, baseTask *utils.Task, curPhase int) *TaskPhase2{
	return &TaskPhase2{pipeCtx: pipeCtx, img: img, kernels: kernels, baseTask: baseTask, curPhase: curPhase}
}

// Apply the effects in `kernels` to the image `img`.
// If nSubThreads == 1, the `Worker` thread itself will apply the effects.
// If nSubThreads > 1, the `Worker` thread will slice the image and spawn `nSubThreads` to process the slices.
func (t2 *TaskPhase2) Execute(wID int){
	// nSubThreads > 1 => slice the image and spawn sub-threads to process the slices
	nSubThreads := t2.pipeCtx.config.SubThreadCount
	if nSubThreads > 1 {
		// create slices of the image
		imgSlices := SlicesByRow(t2.img, nSubThreads)
		
		// constructs to synchronize sub-threads
		sCtx := NewSyncContext(nSubThreads)
		sCtx.wg.Add(len(imgSlices))

		// spawn subthreads to process each slice 
		for _, imgSlice := range imgSlices {
			go  applyManyThreads(t2.img, imgSlice, t2.kernels, sCtx)
		}

		// wait for all subthreads to finish their slices
		sCtx.wg.Wait()
	
	// nSubThreads == 1 => apply effects in 'kernels' to the image 'img' in this thread
	} else {
		applyOneThread(t2.img, t2.kernels)
	}
	
	// create task for phase 3 with results and send to channel
	taskPhase3 := NewTaskPhase3(t2.pipeCtx, t2.baseTask, t2.img, t2.curPhase+1)
	t2.pipeCtx.channels[t2.curPhase+1] <- taskPhase3

	// signalize this task is done to the go-routine managing the overall pipeline
	t2.pipeCtx.wgs[t2.curPhase].Done()
}

// Apply all effects in 'kernels to a slice of 'img'. Each sub-thread waits for
// for other sub-threads to finish the application of an effect before proceeding to the next effect.
func applyManyThreads(img *png.Image, slice ImageSlice, kernels []*png.Kernel, ctx *syncContext) {
   
	// loop: apply each effect in 'kernels' to the image slice
   for _, kernel := range kernels {
	   // apply effect
	   img.ApplyEffectSlice2(kernel, slice.YStart, slice.YEnd, slice.XStart, slice.XEnd)

	   // Barrier: waits for the other threads to finish current effect before proceeding to the next. 
	   // If last thread, reset counter, invert buffer and signal threads can start next effect.
	   ctx.mutex.Lock()
	   ctx.counter++
	   if ctx.counter == ctx.nThreads {
			ctx.counter = 0
			// invert image buffer for application of next effect (see png.Image struct definition)
			img.Final = 1 - img.Final
			ctx.cond.Broadcast()
	   } else {
			ctx.cond.Wait()
	   }
	   ctx.mutex.Unlock()
	}
	// signal slice processing complete
	ctx.wg.Done()
}

// Apply all effects in 'kernels to the image 'img'.
func applyOneThread(img *png.Image, kernels []*png.Kernel) {
	for _, kernel := range kernels {
		img.ApplyEffect(kernel)
		// invert image buffer for application of next effect (see png.Image struct definition)
		img.Final = 1 - img.Final
	}
}

// Not used; just to implement the `ws.Runnable` interface.
func(t2 *TaskPhase2) GetTaskID() int{return 0}

//=============================================================================
// Phase 3: Save images
//=============================================================================

// TaskPhase3 implements `ws.Runnable` interface.
// Each image to be saved is associated to a `TaskPhase3`.
type TaskPhase3 struct {
	pipeCtx 		*PipeContext	  // parameters of the overall pipeline
	baseTask 		*utils.Task		  // contains info of the image to be saved. Ex: outPath
	img 			*png.Image		  // final image to be saved
	curPhase 		int				  // pipeline phase this task belongs to
}

func NewTaskPhase3(pipeCtx *PipeContext, baseTask *utils.Task, img *png.Image, curPhase int) *TaskPhase3{
	return &TaskPhase3{pipeCtx: pipeCtx, baseTask: baseTask, img: img, curPhase: curPhase}
}

// Save the image to disk and signalize main routine the task is done.
func (t3 *TaskPhase3) Execute(wID int){
	// fmt.Println("Saving image: ", t3.baseTask.OutPath)
	t3.img.Save(t3.baseTask.OutPath)

	// signalize this task is done to the go-routine managing the overall pipeline
	t3.pipeCtx.wgs[t3.curPhase].Done()
}

// Not used; just to implement the `ws.Runnable` interface.
func(t3 *TaskPhase3) GetTaskID() int{return 0}

