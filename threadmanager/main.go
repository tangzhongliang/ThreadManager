package main

import (
	"fmt"
	"threadmanager/controllers"
	"time"
)

func main() {
	tm := controllers.ThreadManager{}
	var threadWorkers []controllers.ThreadWorker
	for i := 0; i < 10; i++ {
		threadWorkers = append(threadWorkers, controllers.CreateThreadWorker(fmt.Sprintf("type%d", i), 100, print))
	}
	tm.Init(threadWorkers)
	tm.Start()
	var msg *controllers.ThreadMessage
	for i := 0; i < 200; i++ {
		if i < 100 {
			//工作池处理下载任务
			tm.ExecuteMessage(tm.CreateThreadMessage("type1", "", fmt.Sprintf("%d", i), nil))
		} else if i < 200 {
			//对会议室资源,奇偶数串行化执行操作
			msg = tm.CreateThreadMessage("type2", fmt.Sprintf("%d", i%2), fmt.Sprintf("%d", i), nil)
			tm.ExecuteMessage(msg)
			if msg.Value == "190" {
				msg.Cancel()
			}
		}
	}
	fmt.Printf("result%s", msg.GetResult())
	fmt.Printf("result%s", msg.GetResult())
	fmt.Printf("result%s", msg.GetResult())
	fmt.Printf("result%s", msg.GetResult())
	fmt.Printf("result%s", msg.GetResult())
	time.Sleep(1 * time.Minute)
}
func print(workerType string, value interface{}) interface{} {
	fmt.Printf("%s:%s  ", workerType, value.(string))
	time.Sleep(5 * time.Microsecond)
	return value
}
