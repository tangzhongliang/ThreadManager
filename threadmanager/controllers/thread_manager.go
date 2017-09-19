/*
thread manager is used to
*/
package controllers

import (
	"container/list"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

type ThreadManager struct {
	syncListMap      map[string]*ListSync
	threadWorkers    map[string]*ThreadWorker
	messageSubscribe chan *ThreadMessage
	resultSubscribe  chan *ThreadMessage
	started          bool
	ChanLength       int
	quit             chan bool
}

type ThreadWorkerBase struct {
	resultSubscribe chan *ThreadMessage
	list            *ListSync
	tasks           chan *ThreadTask
	taskCount       int
	running         bool
}
type ThreadWorker struct {
	ThreadWorkerBase
	WorkerType     string
	WorkerSize     int
	DoInBackground func(workerType string, v interface{}) interface{}
}

type ThreadTask struct {
	Job *ThreadMessage
	Tm  *ThreadManager
}

const (
	Waiting    = 1
	Processing = 2
	Completed  = 3
	Canceled   = 4
)

type ThreadMessage struct {
	Id             string
	WorkerType     string
	ValueSyncKey   string
	Value          interface{}
	DoInBackground func(workerType string, v interface{}) interface{}
	Callback       func(workerType string, v interface{})
	cancel         bool
	waitChan       chan interface{}
	Result         interface{}
	state          int
}

func (this *ThreadMessage) GetResult() interface{} {
	if this.waitChan == nil {
		this.waitChan = make(chan interface{})
	}
	tmp, ok := <-this.waitChan
	if tmp != nil && ok {
		this.Result = tmp
		close(this.waitChan)
	}
	return this.Result
}
func (this *ThreadMessage) Cancel() bool {
	if this.state == Waiting {
		this.cancel = true
		return true
	} else {
		return false
	}
}

func (this *ThreadMessage) GetState() int {
	return this.state
}

/*
fengzhuang list
*/
type ListSync struct {
	List *list.List
}

func newListSync() *ListSync {
	return &ListSync{list.New()}
}
func (this *ListSync) len() int {
	return this.List.Len()
}
func (this *ListSync) pop() *list.Element {
	el := this.List.Front()
	this.List.Remove(el)
	return el
}
func (this *ListSync) push(v interface{}) {
	this.List.PushBack(v)
}

func CreateThreadWorker(workerType string, workerSize int, doin func(workerType string, v interface{}) interface{}) ThreadWorker {
	return ThreadWorker{WorkerType: workerType, WorkerSize: workerSize, DoInBackground: doin}
}

func (this *ThreadWorker) ExecuteTask(tsk *ThreadTask) {
	this.tasks <- tsk
}

func (this *ThreadWorker) WorkerExecuteTask(job *ThreadMessage, tm *ThreadManager) {
	//execute task
	//return result to block call
	//execute task callback
	if !job.cancel {
		job.state = Processing
		tmp := job.DoInBackground(job.WorkerType, job.Value)
		if job.waitChan != nil {
			job.waitChan <- job.Value
		}
		job.state = Completed
		if job.Callback != nil {
			job.Callback(job.WorkerType, tmp)
		}
	} else {
		job.state = Canceled
		if job.waitChan != nil {
			close(job.waitChan)
		}
	}

	defer func() {
		//notify woker that job finished
		//sync key not null ,then notify manager that job finished
		this.resultSubscribe <- job
		if len(job.ValueSyncKey) > 0 {
			tm.resultSubscribe <- job
		}
	}()
}

func (this *ThreadWorker) start() error {
	if this.running {
		err := errors.New("thread worker has started")
		panic(err)
		return err
	}

	this.resultSubscribe = make(chan *ThreadMessage, this.WorkerSize)
	this.tasks = make(chan *ThreadTask, this.WorkerSize)
	this.list = newListSync()
	this.resultSubscribe = make(chan *ThreadMessage, this.WorkerSize)

	this.running = true
	stop := make(chan bool)
	//channel init size
	go func(this *ThreadWorker) {
		stop <- true
		for {
			select {
			case <-this.resultSubscribe:
				//task list >0 get next task to execute
				//task list =0 reduce task count

				if this.list.len() > 0 {
					//get next task
					tsk := this.list.pop().Value.(*ThreadTask)
					go this.WorkerExecuteTask(tsk.Job, tsk.Tm)
				} else {
					// taskCount -1
					this.taskCount--
				}
			case tsk := <-this.tasks:
				//task list >0 or task count > size push task into list
				//otherwise execute task

				if this.list.len() > 0 || this.taskCount > this.WorkerSize {
					this.list.push(tsk)
				} else {
					this.taskCount++
					go this.WorkerExecuteTask(tsk.Job, tsk.Tm)
				}
			}
		}
	}(this)
	<-stop
	return nil
}

func (this *ThreadManager) CreateThreadMessage(workerType, key string, value interface{}, doin func(workerType string, v interface{}) interface{}) *ThreadMessage {
	if doin == nil {
		threadWorker, _ := this.threadWorkers[workerType]
		return &ThreadMessage{WorkerType: workerType, ValueSyncKey: key,
			Value: value, DoInBackground: threadWorker.DoInBackground}
	} else {
		return &ThreadMessage{WorkerType: workerType, ValueSyncKey: key,
			Value: value, DoInBackground: doin}
	}

}
func (this *ThreadManager) ExecuteMessage(msg *ThreadMessage) {
	//todo add random
	msg.Id = strconv.FormatInt(time.Now().UnixNano(), 32) + strconv.Itoa(rand.Int())
	this.messageSubscribe <- msg
}
func (this *ThreadManager) Init(threadWorkers []ThreadWorker) {
	this.syncListMap = make(map[string]*ListSync)
	if this.ChanLength == 0 {
		this.ChanLength = 100000
	}
	this.messageSubscribe = make(chan *ThreadMessage, this.ChanLength)
	this.resultSubscribe = make(chan *ThreadMessage, this.ChanLength)
	this.threadWorkers = make(map[string]*ThreadWorker)
	for _, value := range threadWorkers {
		if value.WorkerSize <= 0 || value.WorkerType == "" {
			panic("worker size must be >0 and worker type must be not null")
		}
		temp := value
		this.threadWorkers[value.WorkerType] = &temp
	}

}

func (this *ThreadManager) Stop() {
	this.quit <- true
}
func (this *ThreadManager) Start() error {
	if this.started {
		err := errors.New("thread manager has started")
		panic(err)
		return err
	}
	this.started = true
	this.quit = make(chan bool)
	for _, worker := range this.threadWorkers {
		//		cmutil.LogI(worker)
		worker.start()
	}

	executeNextSyncTask := func(job *ThreadMessage, this *ThreadManager) {
		//get next task witch sync key is same as sync list
		lst, ok := this.syncListMap[job.ValueSyncKey]
		if ok {
			//lock rename is running
			if lst.len() > 0 {
				tmsg := lst.pop().Value.(*ThreadMessage)

				threadWorker, _ := this.threadWorkers[job.WorkerType]
				pushTaskIntoWorker(threadWorker, tmsg, this)
			} else {
				delete(this.syncListMap, job.ValueSyncKey)
			}
		}
	}
	go func() {
		//todo select sort
		for {
			select {
			case <-this.quit:
				return
			case job := <-this.resultSubscribe:
				executeNextSyncTask(job, this)
			case job := <-this.messageSubscribe:
				// job is sync : put job into sync list
				// sync list is null: execute task
				// job is not sync : execute task
				if job.Value == "199" {
					fmt.Printf("jobbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb%p", job)
				}
				threadWorker, ok := this.threadWorkers[job.WorkerType]
				if !ok || threadWorker == nil {
					panic("message workertype is not correct;" + job.WorkerType)
				}
				if job.ValueSyncKey == "" {
					pushTaskIntoWorker(threadWorker, job, this)
				} else {
					lst, ok := this.syncListMap[job.ValueSyncKey]
					if ok {
						lst.push(job)
					} else {
						newLst := newListSync()
						this.syncListMap[job.ValueSyncKey] = newLst
						pushTaskIntoWorker(threadWorker, job, this)
					}
				}

			}
		}
	}()
	return nil
}

func pushTaskIntoWorker(threadWorker *ThreadWorker, job *ThreadMessage, tm *ThreadManager) {
	threadWorker.ExecuteTask(&ThreadTask{Job: job, Tm: tm})
}
