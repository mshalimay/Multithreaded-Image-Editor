package utils

import(
	"proj3/mysync"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	cons "proj3/constants"
)

type Queue struct {
	
}




// Task is a struct containing the information needed to process an image
// It is used both to parse the user input and as a task queue element to be processed by workers.
// @inPath: path to the input image
// @outPath: path to the output image
// @effects: list of effects to be applied to the image
// reference: using tags to parse JSON https://pkg.go.dev/encoding/json#Marshal
type Task struct {
	InPath  string   `json:"inPath"`
	OutPath string   `json:"outPath"`
	Effects []string `json:"effects"`
}

// TaskQueue is a struct containing a list of tasks and a TASLock to synchronize access to them
// @Tasks: list of `Task` structs to be processed by workers
// @TASLock: test and set lock to synchronize access to the list of tasks

// Obs: for the sake of symmetry and code reutilization, in this project the queue can also be accessed 
// in non-thread safe mode by refering to the Tasks field directly. This way the sequential version
// can use the same data structure as the parallel version (although without sync overhead).
type TaskQueue struct{
	mysync.TASLock
	Tasks []Task
}

// creates and initialize a new TaskQueue struct and returns a pointer to it
func NewTaskQueue() *TaskQueue {
    return &TaskQueue{
        TASLock: mysync.NewTasLock(),
        Tasks:   make([]Task, 0),
    }
}

// Enqueue adds a new task to the queue in thread safe manner
func (tq *TaskQueue) Enqueue(task Task) {
	tq.Lock()
	tq.Tasks = append(tq.Tasks, task)
	tq.Unlock()
}

// Dequeue removes the first Task of the queue in thread safe manner and return a pointer to it
func (tq *TaskQueue) Dequeue() *Task {
	tq.Lock()
	if len(tq.Tasks) > 0 {
		task := (tq.Tasks)[0]
		tq.Tasks = (tq.Tasks)[1:]
		tq.Unlock()
		return &task	
	}
	tq.Unlock()
	return nil
}

// Combines data directories from CMD inputs and effects.txt file
//  to create a queue of tasks and returns a pointer to it.
func CreateTasks(dataDirs string) *TaskQueue {
	// open effects.txt file and instantiate JSON decoder to parse it
	effectsFile, err := os.Open(cons.EffectsPathFile)
	if err != nil{
		fmt.Println("Error opening effects.txt file:", err)
		os.Exit(1)
	}
	defer effectsFile.Close()

	// Split the dataDirs input into individual directories
	// e.g. "s+b" -> ["s", "b"]
	dirs := strings.Split(dataDirs, "+")

	// instantiate JSON decoder to parse effects.txt file
	decoder := json.NewDecoder(effectsFile)

	// queue to populate with Task structs
	tqueue := NewTaskQueue()
	
	// loop over parse effects.txt entries and create new tasks combining with data directories
	for {
		var task Task
		// retrieve next entry from effects.txt file
		// Obs: the Task struct defines the fields to be parsed from the JSON file
		if err := decoder.Decode(&task); err != nil {
			if err.Error() == "EOF" {
				// end of file reached, stop parsing
				break
			} else {
				fmt.Println("Error decoding effects file:", err)
				os.Exit(1)
			}
		}
		// loop over data directories and create a new task for each one
		for _, dir := range dirs {
			// Create a new task with updated paths for each directory
			newTask := Task{
						InPath:  cons.InDir + "/" + dir + "/" + task.InPath,
						OutPath: cons.OutDir + "/" + dir + "_" + task.OutPath,
						Effects: task.Effects,}

			// add new task to the queue
			tqueue.Tasks = append(tqueue.Tasks, newTask)
		}
	}
	return tqueue
}


// Writes 'text' to 'filename', appending to a new line. If the file does not exist, it is created.
func WriteToFile(filename string, text string) {
	
	// try to open the file; create it if it does not exist; open in append mode
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open or create the file: ", err)
		return
	}
	defer file.Close()

	// write 'text' to the file
	_, err = file.WriteString(text)
	if err != nil {
		fmt.Println("Failed to write to the file: ", err)
	}
}

// Prints the current working directory; used for debugging
func PrintWorkingDirectory(){
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	fmt.Println("Current directory is:", dir)
}