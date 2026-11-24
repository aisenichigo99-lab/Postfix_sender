package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Status string

const (
	Queued    Status = "queued"
	Running   Status = "running"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
	Canceled  Status = "canceled"
)

func (s Status) Terminal() bool {
	return s == Succeeded || s == Failed || s == Canceled
}

type Event struct {
	Time     time.Time `json:"time"`
	Status   Status    `json:"status"`
	Progress int       `json:"progress"`
	Message  string    `json:"message"`
}

type Job struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    Status    `json:"status"`
	Progress  int       `json:"progress"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Logs      []Event   `json:"logs"`
}

type Runner func(
	ctx context.Context,
	report func(progress int, message string),
) error

type Subscription struct {
	Snapshot []Event
	Events   <-chan Event
	Cancel   func()
}

type Manager struct {
	mu          sync.RWMutex
	jobs        map[string]*Job
	cancel      map[string]context.CancelFunc
	subscribers map[string]map[chan Event]struct{}
	semaphore   chan struct{}
}

func NewManager() *Manager {
	return &Manager{
		jobs:        make(map[string]*Job),
		cancel:      make(map[string]context.CancelFunc),
		subscribers: make(map[string]map[chan Event]struct{}),
		semaphore:   make(chan struct{}, 1),
	}
}

func newID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

func (m *Manager) Submit(jobType string, runner Runner) (*Job, error) {
	if jobType == "" || runner == nil {
		return nil, errors.New("invalid job")
	}

	id, err := newID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job := &Job{
		ID:        id,
		Type:      jobType,
		Status:    Queued,
		CreatedAt: now,
		UpdatedAt: now,
		Logs:      []Event{},
	}

	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.jobs[id] = job
	m.cancel[id] = cancel
	result := cloneJob(job)
	m.mu.Unlock()

	go m.execute(ctx, id, runner)

	return result, nil
}

func (m *Manager) execute(ctx context.Context, id string, runner Runner) {
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		m.update(id, Canceled, 0, "Task canceled before execution")
		m.cleanup(id)
		return
	}

	if ctx.Err() != nil {
		m.update(id, Canceled, 0, "Task canceled before execution")
		m.cleanup(id)
		return
	}

	m.update(id, Running, 0, "Task started")

	report := func(progress int, message string) {
		if progress < 0 {
			m.log(id, message)
			return
		}

		if progress > 100 {
			progress = 100
		}

		m.update(id, Running, progress, message)
	}

	err := runner(ctx, report)

	switch {
	case ctx.Err() != nil:
		m.update(id, Canceled, 0, "Task canceled")
	case err != nil:
		m.update(id, Failed, 0, err.Error())
	default:
		m.update(id, Succeeded, 100, "Task completed successfully")
	}

	m.cleanup(id)
}

func (m *Manager) update(
	id string,
	status Status,
	progress int,
	message string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return
	}

	event := Event{
		Time:     time.Now().UTC(),
		Status:   status,
		Progress: progress,
		Message:  message,
	}

	job.Status = status
	job.Progress = progress
	job.UpdatedAt = event.Time
	job.Logs = append(job.Logs, event)

	for subscriber := range m.subscribers[id] {
		select {
		case subscriber <- event:
		default:
			// Un abonné trop lent ne bloque jamais une tâche système.
		}
	}
}

func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}

	return cloneJob(job), true
}

func (m *Manager) Subscribe(id string) (*Subscription, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}

	ch := make(chan Event, 32)

	if m.subscribers[id] == nil {
		m.subscribers[id] = make(map[chan Event]struct{})
	}
	m.subscribers[id][ch] = struct{}{}

	snapshot := append([]Event(nil), job.Logs...)

	return &Subscription{
		Snapshot: snapshot,
		Events:   ch,
		Cancel: func() {
			m.mu.Lock()
			defer m.mu.Unlock()

			if subs := m.subscribers[id]; subs != nil {
				delete(subs, ch)
				if len(subs) == 0 {
					delete(m.subscribers, id)
				}
			}
		},
	}, true
}

func (m *Manager) Cancel(id string) bool {
	m.mu.RLock()
	cancel, ok := m.cancel[id]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	cancel()
	return true
}

func (m *Manager) cleanup(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancel, id)
}

func cloneJob(job *Job) *Job {
	copy := *job
	copy.Logs = append([]Event(nil), job.Logs...)
	return &copy
}

func (m *Manager) log(id string, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return
	}

	event := Event{
		Time:     time.Now().UTC(),
		Status:   job.Status,
		Progress: job.Progress,
		Message:  message,
	}

	job.UpdatedAt = event.Time
	job.Logs = append(job.Logs, event)

	for subscriber := range m.subscribers[id] {
		select {
		case subscriber <- event:
		default:
		}
	}
}
