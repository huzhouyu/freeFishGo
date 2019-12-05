package main

import (
	"freeFishGo/router"
	"time"
)

type ctrTest struct {
	router.Controller
}

func (c *ctrTest) GetControllerActionInfo() []*router.ControllerActionInfo {
	println("不是默认GetControllerInfo")
	tmp := make([]*router.ControllerActionInfo, 0)
	tmp = append(tmp, &router.ControllerActionInfo{RouterPattern: "/{ Controller}/{Action}/{id:int}{string}/{int}", ControllerActionFuncName: "MyControllerActionStrut"})
	return tmp
}

type Test struct {
	T  []string `json:"tt"`
	T1 string   `json:"tstst1"`
	Id string   `json:"id"`
}

func (c *ctrTest) MyControllerActionStrut(Test *Test) {
	c.HttpContext.Response.Write([]byte(Test.Id))
}
func main() {
	app := NewFreeFish()
	// 注册控制器
	app.AddHanlers(&ctrTest{})
	// 注册主路由
	app.AddMainRouter(&router.ControllerActionInfo{RouterPattern: "/{ Controller}/{Action}", ControllerActionFuncName: "MyControllerActionStrut"})
	app.Run()
	time.Sleep(time.Hour)
}
