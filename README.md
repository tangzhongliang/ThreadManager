# ThreadManager
## 功能
- 线程任务可以指定回调函数，也可以阻塞获取返回值
- 可以创建多个线程池，并且指定同一时间的最大线程，限制任务资源占有突然增高影响系统总体性能
- 线程任务可以指定一个key,同一个Key的任务保证串行化执行
## 使用介绍
'''
tm := controllers.ThreadManager{}
var threadWorkers []controllers.ThreadWorker
threadWorkers = append(threadWorkers, controllers.CreateThreadWorker("worker", 100, print))
tm.Init(threadWorkers)
tm.Start()
tm.ExecuteMessage(tm.CreateThreadMessage("worker", "", "hello!",nil))

func print(workerType string, value interface{}) interface{} {
	fmt.Printf("%s:%s  ", workerType, value.(string))
	time.Sleep(5 * time.Microsecond)
	return value
}
'''