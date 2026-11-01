package worker

import "context"

type Worker struct {
	executor Executor
	queue    chan Request

	cancel context.CancelFunc
	done   chan struct{}
}

func NewWorker(
	parent context.Context,
	executor Executor,
	queueSize int,
) *Worker {
	ctx, cancel := context.WithCancel(parent)

	w := &Worker{
		executor: executor,
		queue:    make(chan Request, queueSize),
		cancel:   cancel,
		done:     make(chan struct{}),
	}

	go w.run(ctx)

	return w
}

func (w *Worker) run(ctx context.Context) {
	defer close(w.done)

	for {
		select {
		case <-ctx.Done():
			return

		case req := <-w.queue:
			w.execute(req)
		}
	}
}

func (w *Worker) execute(req Request) {
	// キュー待ちの間に呼び出し元がキャンセルされた場合。
	if err := req.Context.Err(); err != nil {
		sendResponse(req, Response{
			Err: err,
		})
		return
	}

	result, err := w.executor.Execute(
		req.Context,
		req.Command,
	)

	sendResponse(req, Response{
		Result: result,
		Err:    err,
	})
}

func sendResponse(req Request, resp Response) {
	select {
	case req.Reply <- resp:
	case <-req.Context.Done():
		// 呼び出し側がもう待っていないなら捨てる。
	}
}

func (w *Worker) Submit(ctx context.Context, req Request) error {
	select {
	case w.queue <- req:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) Close() {
	w.cancel()
	<-w.done
}
