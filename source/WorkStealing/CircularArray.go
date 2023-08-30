package workstealing


// `Runnable` is an interface for generic tasks that can execute themselves
type Runnable interface{
	Execute(wID int)	// Passing the id of the thread executing is useful for debugging, but not necessary.
	GetTaskID() int		// Useful for debugging; not necessary.
}

// `CircularArray` holds tasks that can be accessed by multiple workers using modular arithmetic
type CircularArray struct {
	logCapacity int		// log of the capacity of the circular array. Eg: `logCapacity`=3 => capacity=8
	tasks 		[]Runnable  
}

func NewCircularArray(logCapacity int) *CircularArray {
	return &CircularArray{logCapacity: logCapacity, tasks: make([]Runnable, 1 << logCapacity)}
}


// GetCapacity returns the capacity of the circular array
func (c *CircularArray) GetCapacity() int {
	return 1 << c.logCapacity // 2^logCapacity
}

// GetTask returns the task at the given index
func (c *CircularArray) GetTask(i int) Runnable {
	// mod operator is used because the array may be resized and `top` only increases;
	// this allow the thief to access the task with same indexes before the resize.
	// eg: if capacity 8 a thief may access entry 3 holding `top=3` or `top=11`
	
	// eg: if capacity increases to 16, thief may still access entry 3 with `top=3`; 
	// entry 11 will point to different place but this is the desired behavior:
	// the array is larger and the thief is accessing the proper entry.

	return c.tasks[i % c.GetCapacity()]
}

// PutTask inserts a `Task` in the circular array
func (c *CircularArray) PutTask(i int, task Runnable) {
	// mod operator is used because the array may be resized; this allow threads
	// to insert a task with same indexes before the resize.
	c.tasks[i % c.GetCapacity()] = task
}

// Resize resizes the circular array and transfers the tasks from the old array to the new one
func (c *CircularArray) Resize(bottom, top int) *CircularArray{
	// create a new circular array with double the capacity of the current one
	newCArray := NewCircularArray(c.logCapacity + 1)
	
	// transfer the tasks from the old array to the new one
	for i := top; i < bottom; i++ {
		newCArray.PutTask(i, c.GetTask(i))
	}
	return newCArray
}