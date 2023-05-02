package workpool

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type WorkPool struct {
	Name         string
	ch           chan Job
	workersCount int

	// Для синхронизации старта\завершения воркеров
	wgStart     sync.WaitGroup
	wgFinish    sync.WaitGroup
	wgJobsQueue sync.WaitGroup

	// Для корректного завершения workpool
	finishCh       chan struct{}
	unfinishedJobs []Job
	lockerSave     sync.Mutex
}

func NewWorkPool(name string, workersCount int) (*WorkPool, error) {
	if workersCount < 1 {
		return nil, fmt.Errorf("workers count can't be less than 1")
	}
	return &WorkPool{Name: name, workersCount: workersCount}, nil
}

func (wp *WorkPool) Init(chBufferSize int) error {
	// определение буф.канала для задач
	if chBufferSize < 1 {
		return fmt.Errorf("buffer size can't be less than 1")
	}
	wp.ch = make(chan Job, chBufferSize)
	// инициализация контекста
	wp.finishCh = make(chan struct{})

	// определение waitGroup для запуска и завершения воркеров
	wp.wgStart, wp.wgFinish = sync.WaitGroup{}, sync.WaitGroup{}
	wp.wgStart.Add(wp.workersCount)
	wp.wgFinish.Add(wp.workersCount)

	// создание и запуск воркеров
	for i := 0; i < wp.workersCount; i++ {
		go func(workerIndex int) {
			wp.wgStart.Done()        // сигнал о том, что горутина-воркер запустилась
			defer wp.wgFinish.Done() // сигнал о том, что горутина закончила работу

			// выполнение задачи
			for job := range wp.ch {
				log.Printf("worker with index '%d' used", workerIndex)
				job.Action()
			}
		}(i)
	}
	wp.wgStart.Wait()
	return nil
}

func (wp *WorkPool) Close() {
	// посылаем сигнал в горутины зависшие в ожидании(ф-ия CreateJob) и ждем пока очередь освободится
	close(wp.finishCh)
	wp.wgJobsQueue.Wait()

	close(wp.ch)
	wp.wgFinish.Wait()
}

func (wp *WorkPool) CreateJob(job Job, ctx context.Context) {
	wp.wgJobsQueue.Add(1)
	select {
	// если канал finishCh закрыт - записывать все новые задачи в список незаверш и освобожд. ф-ии CreateJob
	case <-wp.finishCh:
		wp.lockerSave.Lock()
		log.Println("workpool closing, job returned")
		wp.unfinishedJobs = append(wp.unfinishedJobs, job)
		wp.lockerSave.Unlock()
	// если канал для работ доступен для записи
	case wp.ch <- job:
		log.Println("job sent into workpool channel")
	// если контекст вызвавший CreateJob, вызвал отмену контекста(работает!)
	case <-ctx.Done():
		log.Println("job was canceled by context")
	}
	wp.wgJobsQueue.Done()
}
