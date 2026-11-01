package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrWorkerNotFound = errors.New("worker not found")

type WorkerManager struct {
	mu      sync.RWMutex
	workers map[TargetID]*Worker
}

func NewWorkerManager() *WorkerManager {
	return &WorkerManager{
		workers: make(map[TargetID]*Worker),
	}
}

func (m *WorkerManager) Register(
	targetID TargetID,
	worker *Worker,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workers[targetID] = worker
}

func (m *WorkerManager) Unregister(
	targetID TargetID,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.workers, targetID)
}

func (m *WorkerManager) Execute(
	ctx context.Context,
	targetID TargetID,
	command Command,
) (Result, error) {
	worker, err := m.findWorker(targetID)
	if err != nil {
		return nil, err
	}

	reply := make(chan Response, 1)

	req := Request{
		Context: ctx,
		Command: command,
		Reply:   reply,
	}

	if err := worker.Submit(ctx, req); err != nil {
		return nil, err
	}

	select {
	case resp := <-reply:
		return resp.Result, resp.Err

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *WorkerManager) findWorker(
	targetID TargetID,
) (*Worker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	worker, ok := m.workers[targetID]
	if !ok {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrWorkerNotFound,
			targetID,
		)
	}

	return worker, nil
}
