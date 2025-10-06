package app

import (
	"github.com/LagrangeDev/LagrangeGo/client"
	"github.com/LagrangeDev/LagrangeGo/client/auth"
	"github.com/vintcessun/XMU-CM-Bot/bot"
	"github.com/vintcessun/XMU-CM-Bot/config"
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/tools"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

// Container 依赖注入容器
type Container struct {
	config       *config.Config
	logger       *utils.ProtocolLogger
	client       *client.QQClient
	bot          *bot.Bot
	logicManager *event.LogicManager
}

// NewContainer 创建新的容器实例
func NewContainer() *Container {
	return &Container{}
}

// Initialize 初始化所有依赖
func (c *Container) Initialize() error {
	c.config = &config.Config{}
	c.logger = utils.GetProtocolLogger()

	// 初始化配置
	config.Init()
	c.config = config.GlobalConfig

	// 初始化日志
	utils.Init()

	// 创建客户端
	appInfo := auth.AppList["linux"]["3.2.19-39038"]
	c.client = client.NewClient(c.config.Bot.Account, c.config.Bot.Password)
	c.client.SetLogger(c.logger)
	c.client.UseVersion(appInfo)
	c.client.AddSignServer(c.config.Bot.SignServer)
	c.client.UseDevice(auth.NewDeviceInfo(114514))

	// 创建Bot
	c.bot = bot.NewBot(c.client)

	// 加载签名文件
	c.bot.GetAuthManager().LoadSig()

	// 创建逻辑管理器
	c.logicManager = event.NewLogicManager(c.client)

	// 预加载功能
	err := tools.Initialize(c.config, c.logger)
	if err != nil {
		c.logger.Error("预加载错误：", err)
	}

	return nil
}

// GetBot 获取Bot实例
func (c *Container) GetBot() *bot.Bot {
	return c.bot
}

// GetLogicManager 获取逻辑管理器实例
func (c *Container) GetLogicManager() *event.LogicManager {
	return c.logicManager
}

// GetClient 获取客户端实例
func (c *Container) GetClient() *client.QQClient {
	return c.client
}

// GetConfig 获取配置实例
func (c *Container) GetConfig() *config.Config {
	return c.config
}
