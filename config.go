package ipstack

// WorkerPoolConfig represents a configuration for an ipstack.WorkerPool
type WorkerPoolConfig struct {
	QueueSize int
	Workers   int
	Log       Logger
}

// NewDefaultWorkerPoolConfig returns a new ipstack.WorkerPoolConfig populated
// with default values and a dev/null logger implementation
func NewDefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		QueueSize: 1000,
		Workers:   4,
		Log:       devNullLogger{},
	}
}
