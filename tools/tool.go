package tools

import (
	"github.com/vintcessun/XMU-CM-Bot/config"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

var Logger *utils.ProtocolLogger

func Initialize(c *config.Config, logger *utils.ProtocolLogger) error {
	Logger = logger

	logger.Info("预加载LLM")
	err := LLMInit(c)
	if err != nil {
		logger.Error("预加载LLM失败")
		return err
	}

	logger.Info("预加载login")
	err = LoginInit(c)
	if err != nil {
		logger.Error("login预加载和同步任务配置失败")
	}

	logger.Info("预加载check_session")
	err = CheckSessionInit()
	if err != nil {
		logger.Error("check_session预加载失败")
	}

	logger.Info("预加载DB")
	err = DBInit(c)
	if err != nil {
		logger.Error("DB预加载失败")
	}

	return nil
}

func DeInitialize() {
	Logger.Info("停止Pixiv任务")

	Logger.Info("停止Login任务")
	err := Login.Stop()
	if err != nil {
		Logger.Error("停止Login任务失败")
	}

	Logger.Info("停止DB任务")
	err = Db.DeInit()
	if err != nil {
		Logger.Error("停止DB任务失败")
	}
}
