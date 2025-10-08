package help

import (
	"github.com/LagrangeDev/LagrangeGo/message"
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/tools"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

func Help(ctx *event.MessageContext) {
	utils.Info("处理help指令")
	defer utils.Info("处理结束help指令")

	msg := ctx.AssertGroupMessage()
	tools.Login.Delete(msg.Sender.Uin)
	ctx.SendMessage([]message.IMessageElement{message.NewText(`帮助信息：
	/login - 登录
	/logout - 登出
	/download - 下载课程文件
	/search - 搜索所有文件根据关键词
	本项目仓库 https://github.com/vintcessun/XMU-CM-Bot`)})
}
