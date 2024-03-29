package goroutine

// thank https://github.com/ivpusic/grpool
import (
	"context"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"runtime/debug"
	"sync"
	"time"
)

// Grouting instance which can accept client jobs
type worker struct {
	workerPool chan *worker
	jobChannel chan Job
	stop       chan struct{}
}

func (w *worker) start() {
	go func() {
		var job Job
		for {
			// worker free, add it to pool
			w.workerPool <- w

			select {
			case job = <-w.jobChannel:
				runJob(job)
			case <-w.stop:
				w.stop <- struct{}{}
				return
			}
		}
	}()
}

func runJob(f func()) {
	defer func() {
		if err := recover(); err != nil {
			if vars.FrameworkLogger != nil {
				vars.FrameworkLogger.Error(context.Background(), "[gPool] runJob panic err %v, stack: %v",
					err, string(debug.Stack()[:]))
			} else {
				logging.Errf("[gPool] runJob panic err %v, stack: %v\n",
					err, string(debug.Stack()[:]))
			}
		}
	}()
	f()
}

func newWorker(pool chan *worker) *worker {
	return &worker{
		workerPool: pool,
		jobChannel: make(chan Job),
		stop:       make(chan struct{}),
	}
}

// Accepts jobs from clients, and waits for first free worker to deliver job
type dispatcher struct {
	workerPool chan *worker
	jobQueue   chan Job
	stop       chan struct{}
}

func (d *dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			worker := <-d.workerPool
			worker.jobChannel <- job
		case <-d.stop:
			for i := 0; i < cap(d.workerPool); i++ {
				worker := <-d.workerPool

				worker.stop <- struct{}{}
				<-worker.stop
			}

			d.stop <- struct{}{}
			return
		}
	}
}

func newDispatcher(workerPool chan *worker, jobQueue chan Job) *dispatcher {
	d := &dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
		stop:       make(chan struct{}),
	}

	for i := 0; i < cap(d.workerPool); i++ {
		worker := newWorker(d.workerPool)
		worker.start()
	}

	go d.dispatch()
	return d
}

// Job Represents user request, function which should be executed in some worker.
type Job func()

type Pool struct {
	JobQueue   chan Job
	dispatcher *dispatcher
	wg         sync.WaitGroup
}

// NewPool Will make pool of gorouting workers.
// numWorkers - how many workers will be created for this pool
// queueLen - how many jobs can we accept until we block
//
// Returned object contains JobQueue reference, which you can use to send job to pool.
func NewPool(numWorkers int, jobQueueLen int) *Pool {
	if numWorkers <= 0 {
		numWorkers = 2
	}
	if jobQueueLen <= 0 {
		jobQueueLen = 5
	}
	jobQueue := make(chan Job, jobQueueLen)
	workerPool := make(chan *worker, numWorkers)

	pool := &Pool{
		JobQueue:   jobQueue,
		dispatcher: newDispatcher(workerPool, jobQueue),
	}

	return pool
}

func (p *Pool) wrapJob(job func()) func() {
	return func() {
		defer p.JobDone()
		job()
	}
}

func (p *Pool) SendJobWithTimeout(job func(), t time.Duration) bool {
	select {
	case <-time.After(t):
		return false
	case p.JobQueue <- p.wrapJob(job):
		p.WaitCount(1)
		return true
	}
}

func (p *Pool) SendJobWithDeadline(job func(), t time.Time) bool {
	s := t.Sub(time.Now())
	if s <= 0 {
		s = time.Second // timeout
	}
	select {
	case <-time.After(s):
		return false
	case p.JobQueue <- p.wrapJob(job):
		p.WaitCount(1)
		return true
	}
}

func (p *Pool) SendJob(job func()) {
	p.WaitCount(1)
	p.JobQueue <- p.wrapJob(job)
}

// JobDone In case you are using WaitAll fn, you should call this method
// every time your job is done.
//
// If you are not using WaitAll then we assume you have your own way of synchronizing.
func (p *Pool) JobDone() {
	p.wg.Done()
}

// WaitCount How many jobs we should wait when calling WaitAll.
// It is using WaitGroup Add/Done/Wait
func (p *Pool) WaitCount(count int) {
	p.wg.Add(count)
}

// WaitAll Will wait for all jobs to finish.
func (p *Pool) WaitAll() {
	p.wg.Wait()
}

// Release Will release resources used by pool
func (p *Pool) Release() {
	p.dispatcher.stop <- struct{}{}
	<-p.dispatcher.stop
}
