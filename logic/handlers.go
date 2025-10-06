package logic

import (
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/logic/download"
	"github.com/vintcessun/XMU-CM-Bot/logic/login"
	"github.com/vintcessun/XMU-CM-Bot/logic/logout"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

func loggerAddHandler(command []string, function func(*event.MessageContext)) {
	for _, cmd := range command {
		event.Manager.HandleCommand("/", cmd, func(ctx *event.MessageContext) error {
			utils.Info("指令内容 ", ctx.GetText())
			if ok := ctx.RejectNotGroupMessage(); ok {
				function(ctx)
			}
			return nil
		})
	}
}

func RegisterCustomLogic() {
	if event.Manager == nil {
		utils.Error("Logicevent.Manager 未初始化")
		return
	}

	loggerAddHandler([]string{"login"}, login.Login)
	loggerAddHandler([]string{"logout"}, logout.Logout)
	loggerAddHandler([]string{"download"}, download.Download)

	utils.Info("自定义逻辑注册完成")
}
