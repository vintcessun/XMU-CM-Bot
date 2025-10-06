package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vintcessun/XMU-CM-Bot/app"
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/logic"
	"github.com/vintcessun/XMU-CM-Bot/tools"
)

func main() {
	// 使用依赖注入容器
	container := app.NewContainer()
	err := container.Initialize()
	if err != nil {
		panic(err)
	}

	bot := container.GetBot()
	logicManager := container.GetLogicManager()

	// 登录
	err = bot.Login()
	if err != nil {
		panic(err)
	}

	// 监听
	bot.Listen()

	// 注册自定义逻辑
	event.Manager = logicManager
	logic.RegisterCustomLogic()

	// 设置事件监听
	logicManager.SetupEventListeners()

	defer bot.Client().Release()
	defer bot.Dumpsig()
	defer tools.DeInitialize()

	// setup the main stop channel
	mc := make(chan os.Signal, 2)
	signal.Notify(mc, os.Interrupt, syscall.SIGTERM)
	for {
		switch <-mc {
		case os.Interrupt, syscall.SIGTERM:
			return
		}
	}

}
