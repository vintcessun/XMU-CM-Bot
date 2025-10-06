package logout

import (
	"github.com/LagrangeDev/LagrangeGo/message"
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/tools"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

func Logout(ctx *event.MessageContext) {
	utils.Info("处理logout指令")
	defer utils.Info("处理结束logout指令")

	msg := ctx.AssertGroupMessage()
	tools.Login.Delete(msg.Sender.Uin)
	ctx.SendMessage([]message.IMessageElement{message.NewText("已删除session")})
}
