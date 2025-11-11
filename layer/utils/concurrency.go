package utils

import (
	"sync"
)

func SafeGoroutine(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logger := GetLogger()
				logger.Error("Panic recovered in goroutine", nil, map[string]interface{}{
					"panic": r,
				})
			}
		}()
		fn()
	}()
}

type WorkerPool struct {
	workerCount int
	jobChan     chan func()
	wg          sync.WaitGroup
	once        sync.Once
}

func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 10
	}
	if queueSize <= 0 {
		queueSize = 100
	}

	wp := &WorkerPool{
		workerCount: workerCount,
		jobChan:     make(chan func(), queueSize),
	}

	// Start workers
	for i := 0; i < workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for job := range wp.jobChan {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger := GetLogger()
					logger.Error("Panic recovered in worker pool", nil, map[string]interface{}{
						"panic": r,
					})
				}
			}()
			job()
		}()
	}
}

func (wp *WorkerPool) Submit(job func()) {
	wp.jobChan <- job
}

func (wp *WorkerPool) Wait() {
	wp.once.Do(func() {
		close(wp.jobChan)
	})
	wp.wg.Wait()
}

func (wp *WorkerPool) Close() {
	wp.Wait()
}
