package text

import (
	"sync"
	"time"
)

// Shot task statuses returned by GET /generate-shots/status/:taskId.
const (
	ShotTaskProcessing = "processing"
	ShotTaskSucceeded  = "succeeded"
	ShotTaskFailed     = "failed"
)

// shotTaskTTL bounds how long a finished task stays available for polling
// before cleanup. Generations can take 5-35 minutes, so a generous margin is
// used.
const shotTaskTTL = 4 * time.Hour

// ShotTask tracks an in-flight background generate-shots breakdown so clients
// can poll for the result instead of holding a request open for the whole
// Claude call.
type ShotTask struct {
	TaskID    string
	Status    string
	Model     string
	Text      string
	Error     string
	CreatedAt time.Time
}

// ShotTaskStore is an in-memory, single-process store for shot generation
// tasks. Background goroutines write the result and the polling handler reads
// it; a periodic cleanup removes expired tasks.
type ShotTaskStore struct {
	mu    sync.Mutex
	tasks map[string]*ShotTask
}

func NewShotTaskStore() *ShotTaskStore {
	s := &ShotTaskStore{tasks: make(map[string]*ShotTask)}
	go s.cleanupLoop()
	return s
}

func (s *ShotTaskStore) Set(t *ShotTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.TaskID] = t
}

func (s *ShotTaskStore) Get(taskID string) (*ShotTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return nil, false
	}
	cp := *t
	return &cp, true
}

// Update applies fn to the stored task atomically (used by background
// goroutines to flip status and store the result).
func (s *ShotTaskStore) Update(taskID string, fn func(*ShotTask)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[taskID]; ok {
		fn(t)
	}
}

// cleanupLoop periodically evicts tasks older than shotTaskTTL.
func (s *ShotTaskStore) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-shotTaskTTL)
		for id, t := range s.tasks {
			if t.CreatedAt.Before(cutoff) {
				delete(s.tasks, id)
			}
		}
		s.mu.Unlock()
	}
}
