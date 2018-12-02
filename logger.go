package ipstack

// Logger defines a simple interface for logging info and error messages that
// occur during a Workers lifetime. The interface is chosen instead of a
// specific logging framework to decouple the package from other external
// dependencies and avoid stdout stderr logging
type Logger interface {
	Info(msg string)
	Error(msg string, err error)
}

type devNullLogger struct{}

func (wl devNullLogger) Info(msg string)             {}
func (wl devNullLogger) Error(msg string, err error) {}
