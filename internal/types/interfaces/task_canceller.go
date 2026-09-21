package interfaces

import "context"

// TaskCanceller stops a task by its queue-level identifier. Implementations
// may remove a pending task or signal an active worker through its context.
type TaskCanceller interface {
	CancelTask(ctx context.Context, queue, taskID string) (bool, error)
}
