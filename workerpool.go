package ipstack

import (
	"fmt"
	"strconv"
	"sync"
)

var (
	// ErrFeedbackExistsFailed occurs when the provided implementation of
	// ipstack.WorkerFeedback.Exists returns an error.
	ErrFeedbackExistsFailed = "ipstack: WorkerPool: unable to check if ip already exists during feedback loop"
	// ErrAPIRequestFailed occurs when the external ipstack api returns an
	// error, or fails to meet the timeout requirements.
	ErrAPIRequestFailed = "ipstack: WorkerPool: error during api request"
	// ErrFeedbackCreateResponseFailed occurs when the provided implementation
	// of ipstack.WorkerFeedback.CreateResponse returns an error.
	ErrFeedbackCreateResponseFailed = "ipstack: WorkerPool: unable to create response during feedback loop"
)

// WorkerFeedback is used to give single workers inside the ipstack.WorkerPool
// feedback about the way they should handle the ip address in question
type WorkerFeedback interface {
	Exists(ip string) (exists bool, err error)
	CreateResponse(ip string, r *Response) (err error)
}

// WorkerPool represents a single ipstack.WorkerPool, which is able to perform
// ipstack IP Checks in a coordinated way across a fleet of goroutine workers.
// The pool increases performance and removes blocking calls to external APIs
// from your own goroutine
type WorkerPool struct {
	unresolved chan string
	shutdown   chan struct{}
	wg         sync.WaitGroup
	fb         WorkerFeedback
	c          *Client

	Config *WorkerPoolConfig
}

// NewWorkerPool initializes a new WorkerPool instance. It performs runtime-
// checks for the passed arguments and starts all worker goroutines.
func NewWorkerPool(config *WorkerPoolConfig, c *Client, fb WorkerFeedback) (wp *WorkerPool, err error) {
	if config == nil {
		return nil, fmt.Errorf("ipstack: unable to create WorkerPool with nil config")
	}
	if config.Log == nil {
		return nil, fmt.Errorf("ipstack: unable to create WorkerPool with nil logger in config")
	}

	if c == nil {
		return nil, fmt.Errorf("ipstack: unbale to create WorkerPool with nil client")
	}

	if fb == nil {
		return nil, fmt.Errorf("ipstack: unable to create WorkerPool with nil WorkerFeeback")
	}

	// allocate WorkerPool
	wp = &WorkerPool{
		unresolved: make(chan string, config.QueueSize),
		shutdown:   make(chan struct{}),
		fb:         fb,
		c:          c,
		Config:     config,
	}

	// start worker goroutines
	for i := 0; i < config.Workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i + 1)
	}

	return wp, nil
}

// Queue queues the passed ip into the internal buffered channel for unresolved
// IPs. Workers will dequeue it once they are ready. If a shutdown occurs
// before all IPs are dequeued, the shutdown caller will synchronously handle
// all remaining IPs
func (wp *WorkerPool) Queue(ip string) {
	wp.unresolved <- ip
}

// Shutdown shutdowns all previously started workers and handles all remaining
// unresolved IPs synchronously before returing
func (wp *WorkerPool) Shutdown() {
	// send shutdown signal
	close(wp.shutdown)

	// wait for all to finish
	wp.wg.Wait()

	// finish remaining IPs in the buffer
	for len(wp.unresolved) > 0 {
		wp.resolveEntry(<-wp.unresolved)
	}
}

func (wp *WorkerPool) worker(i int) {
	wp.Config.Log.Info("ipstack: WorkerPool: worker" + strconv.Itoa(i) + " started")
	for {
		select {
		case <-wp.shutdown:
			wp.Config.Log.Info("ipstack: WorkerPool: worker" + strconv.Itoa(i) + " received shutdown. Finishing")
			wp.wg.Done()
		case ip := <-wp.unresolved:
			wp.resolveEntry(ip)
		}
	}
}

func (wp *WorkerPool) resolveEntry(ip string) {
	// check if ip already exists
	exists, err := wp.fb.Exists(ip)
	if err != nil {
		wp.Config.Log.Error(ErrFeedbackExistsFailed, err)
		return
	}

	// ip already resolved -> nothing to do
	if exists {
		return
	}

	// ip not resolved -> query external API
	r, err := wp.c.Check(ip)
	if err != nil {
		wp.Config.Log.Error(ErrAPIRequestFailed, err)
		return
	}

	// send response to feedback loop
	err = wp.fb.CreateResponse(ip, r)
	if err != nil {
		wp.Config.Log.Error(ErrFeedbackCreateResponseFailed, err)
	}
}
