package pull

import (
	"github.com/LagrangeDev/LagrangeGo/message"
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

func pullFunc(ctx *event.MessageContext) ([]message.IMessageElement, error) {
	msg := ctx.AssertGroupMessage()

	fs, err := ctx.Client.GetGroupFileSystemInfo(msg.GroupUin)
	if err != nil {
		return nil, err
	}
}

func Pull(ctx *event.MessageContext) {
	utils.Info("处理pull指令")
	defer utils.Info("处理结束pull指令")

	_, ok := ctx.AssertGroupAndRejectExpired()
	if !ok {
		return
	}

	result, err := pullFunc(ctx)

	if err != nil {
		utils.Error("处理失败: ", err)
		ctx.SendMessage([]message.IMessageElement{message.NewText("处理失败: " + err.Error())})
		return
	}

	ctx.SendMessage(result)
}
