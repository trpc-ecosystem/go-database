// Package timer is the implement of timer service.
package timer

import (
	"time"
)

// Scheduler timer preemption scheduler interface
type Scheduler interface {
	// distributed timer: task preemption schedule, only one node can preempt successfully at a time.

	// input:
	// serviceName the current service rpc name.
	// newNode the current node address, default is ip:port_pid_timestamp.
	// holdTime the time when the current node wants to hold the task.

	// output:
	// nowNode the address of the successfully preempted node.
	// err preempt successfully or not, only one node can preempt successfully at a time.
	Schedule(serviceName string, newNode string, holdTime time.Duration) (nowNode string, err error)
}

// DefaultScheduler the default local timer scheduler.
type DefaultScheduler struct {
}

// Schedule the local timer has no mutual exclusion, will return the current node directly.
func (s *DefaultScheduler) Schedule(serviceName string,
	newNode string, holdTime time.Duration) (nowNode string, err error) {
	return newNode, nil
}

var schedulers = make(map[string]Scheduler)

// RegisterScheduler register the scheduler
func RegisterScheduler(name string, s Scheduler) {
	schedulers[name] = s
}
