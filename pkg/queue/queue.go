package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type QueuedRequest struct {
	UserID     string
	Payload    string
	Retries    int
	MaxRetries int
	Delay      time.Duration
	Result     chan<- QueueResult
}

type QueueResult struct {
	Success bool
	Error   error
	Data    interface{}
}

type RequestQueue struct {
	queue     chan QueuedRequest
	workers   int
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	processor func(QueuedRequest) QueueResult
}

func NewRequestQueue(workers, queueSize int, processor func(QueuedRequest) QueueResult) *RequestQueue {
	ctx, cancel := context.WithCancel(context.Background())

	rq := &RequestQueue{
		queue:     make(chan QueuedRequest, queueSize),
		workers:   workers,
		ctx:       ctx,
		cancel:    cancel,
		processor: processor,
	}

	rq.start()
	return rq
}

func (rq *RequestQueue) start() {
	for i := 0; i < rq.workers; i++ {
		rq.wg.Add(1)
		go rq.worker()
	}
}

func (rq *RequestQueue) worker() {
	defer rq.wg.Done()

	for {
		select {
		case <-rq.ctx.Done():
			return
		case req := <-rq.queue:
			if req.Delay > 0 {
				select {
				case <-time.After(req.Delay):
				case <-rq.ctx.Done():
					return
				}
			}

			result := rq.processor(req)

			if req.Result != nil {
				select {
				case req.Result <- result:
				case <-rq.ctx.Done():
					return
				}
			}

			if !result.Success && req.Retries < req.MaxRetries {
				retryReq := req
				retryReq.Retries++
				retryReq.Delay = time.Duration(retryReq.Retries) * time.Second

				select {
				case rq.queue <- retryReq:
				case <-rq.ctx.Done():
					return
				}
			}
		}
	}
}

func (rq *RequestQueue) Enqueue(userID, payload string, maxRetries int) <-chan QueueResult {
	resultChan := make(chan QueueResult, 1)

	req := QueuedRequest{
		UserID:     userID,
		Payload:    payload,
		Retries:    0,
		MaxRetries: maxRetries,
		Delay:      0,
		Result:     resultChan,
	}

	select {
	case rq.queue <- req:
	default:
		go func() {
			resultChan <- QueueResult{
				Success: false,
				Error:   fmt.Errorf("queue is full"),
			}
		}()
	}

	return resultChan
}

func (rq *RequestQueue) Stop() {
	rq.cancel()
	rq.wg.Wait()
	close(rq.queue)
}

func (rq *RequestQueue) Size() int {
	return len(rq.queue)
}
