package lane

import (
	"sync"
	"sync/atomic"
)

// WorkerPool manages a pool of goroutines for executing tasks.
type WorkerPool struct {
	maxWorkers int
	taskCh     chan Task
	workerFn   func(Task)
	
	// State
	running    atomic.Bool
	stopCh     chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
	
	// Metrics
	tasksProcessed atomic.Int64
}

// NewWorkerPool creates a new WorkerPool.
func NewWorkerPool(maxWorkers int, workerFn func(Task)) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		taskCh:     make(chan Task),
		workerFn:   workerFn,
		stopCh:     make(chan struct{}),
	}
}

// Start starts the worker pool.
func (p *WorkerPool) Start() {
	if p.running.Load() {
		return
	}
	
	p.running.Store(true)
	
	// Start workers
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully stops the worker pool.
// It waits for all workers to finish processing current tasks.
func (p *WorkerPool) Stop() {
	p.stopOnce.Do(func() {
		p.running.Store(false)
		close(p.stopCh)
		p.wg.Wait()
	})
}

// Submit submits a task to the worker pool.
// This method blocks until a worker is available or the pool is stopped.
func (p *WorkerPool) Submit(task Task) {
	if !p.running.Load() {
		return
	}
	
	select {
	case p.taskCh <- task:
	case <-p.stopCh:
		return
	}
}

// TrySubmit attempts to submit a task without blocking.
// Returns true if the task was submitted, false otherwise.
func (p *WorkerPool) TrySubmit(task Task) bool {
	if !p.running.Load() {
		return false
	}
	
	select {
	case p.taskCh <- task:
		return true
	default:
		return false
	}
}

// worker is the main loop for each worker goroutine.
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	
	for {
		select {
		case task, ok := <-p.taskCh:
			if !ok {
				return
			}
			p.processTask(task)
		case <-p.stopCh:
			// Process remaining tasks in the channel
			for {
				select {
				case task := <-p.taskCh:
					p.processTask(task)
				default:
					return
				}
			}
		}
	}
}

// processTask processes a single task.
func (p *WorkerPool) processTask(task Task) {
	defer func() {
		// Recover from panics to prevent worker crash
		if r := recover(); r != nil {
			// Panic recovered - task failed but worker continues
			// In production, this should be logged via a logger interface
			_ = r
		}
	}()

	p.workerFn(task)
	p.tasksProcessed.Add(1)
}

// TasksProcessed returns the total number of tasks processed.
func (p *WorkerPool) TasksProcessed() int64 {
	return p.tasksProcessed.Load()
}

// IsRunning returns true if the worker pool is running.
func (p *WorkerPool) IsRunning() bool {
	return p.running.Load()
}

// DynamicWorkerPool is a worker pool that can dynamically adjust its size.
type DynamicWorkerPool struct {
	*WorkerPool
	
	minWorkers int
	maxWorkers int
	
	// Metrics for auto-scaling
	idleTime    atomic.Int64
	busyCount   atomic.Int32
	scaleUpCh   chan struct{}
	scaleDownCh chan struct{}
}

// NewDynamicWorkerPool creates a new DynamicWorkerPool.
func NewDynamicWorkerPool(minWorkers, maxWorkers int, workerFn func(Task)) *DynamicWorkerPool {
	return &DynamicWorkerPool{
		minWorkers:  minWorkers,
		maxWorkers:  maxWorkers,
		scaleUpCh:   make(chan struct{}, 1),
		scaleDownCh: make(chan struct{}, 1),
	}
}

// Start starts the dynamic worker pool.
func (p *DynamicWorkerPool) Start() {
	if p.running.Load() {
		return
	}
	
	p.running.Store(true)
	
	// Start minimum number of workers
	for i := 0; i < p.minWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	
	// Start auto-scaling goroutine
	go p.autoScale()
}

// autoScale handles dynamic scaling of workers.
func (p *DynamicWorkerPool) autoScale() {
	currentWorkers := p.minWorkers
	
	for {
		select {
		case <-p.stopCh:
			return
		case <-p.scaleUpCh:
			if currentWorkers < p.maxWorkers {
				p.wg.Add(1)
				go p.worker(currentWorkers)
				currentWorkers++
			}
		case <-p.scaleDownCh:
			if currentWorkers > p.minWorkers {
				// Signal a worker to stop
				// This is a simplified version; in production,
				// you'd want more sophisticated worker lifecycle management
				currentWorkers--
			}
		}
	}
}

// signalScaleUp signals that we need to scale up workers.
func (p *DynamicWorkerPool) signalScaleUp() {
	select {
	case p.scaleUpCh <- struct{}{}:
	default:
	}
}
