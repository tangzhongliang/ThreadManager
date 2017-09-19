package main

import (
	"fmt"
	"threadmanager/controllers"
	"time"
)

func main() {
	var i controllers.ThreadMessage // i 的类型是int型
	i.Value = "123"
	c := make(chan *controllers.ThreadMessage, 1)
	c <- &i
	i.Value = "aaaaa"
	aaa := <-c
	fmt.Println(aaa.Value)
	// beego.Run()
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
			tm.ExecuteMessage(tm.CreateThreadMessage("type1", "", fmt.Sprintf("%d", i)))
		} else if i < 200 {
			//对会议室资源,奇偶数串行化执行操作
			msg = tm.CreateThreadMessage("type2", fmt.Sprintf("%d", i%2), fmt.Sprintf("%d", i))
			tm.ExecuteMessage(msg)
			if msg.Value == "190" {
				fmt.Printf("mainbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb%p,", msg)
				msg.Cancel()
			}
		}
	}
	fmt.Printf("result%s", msg.GetResult())
	time.Sleep(1 * time.Minute)
}
func print(workerType string, value interface{}) interface{} {
	fmt.Printf("%s:%s  ", workerType, value.(string))
	time.Sleep(5 * time.Microsecond)
	return value
}
