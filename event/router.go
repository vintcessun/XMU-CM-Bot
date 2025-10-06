package event

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/LagrangeDev/LagrangeGo/client"
	"github.com/LagrangeDev/LagrangeGo/message"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	message2 "github.com/vintcessun/XMU-CM-Bot/message"
	"github.com/vintcessun/XMU-CM-Bot/tools"
)

// MessageContext 消息上下文
type MessageContext struct {
	Client   *client.QQClient
	Message  message2.Message
	Metadata map[string]interface{}
	ctx      context.Context
	text     string
}

// NewMessageContext 创建新的消息上下文
func NewMessageContext(client *client.QQClient, msg message2.Message) *MessageContext {
	ret := MessageContext{
		Client:   client,
		Message:  msg,
		Metadata: make(map[string]interface{}),
		ctx:      context.Background(),
	}
	ret.InitMessageText()
	return &ret
}

func (mc *MessageContext) InitMessageText() {
	text, ok := mc.GetMessageElements()
	if ok {
		mc.text = extractTextFromElements(text)
	}
}

func (mc *MessageContext) GetMessageElements() ([]message.IMessageElement, bool) {
	if private, ok := mc.GetPrivateMessage(); ok {
		return private.Elements, true
	}
	if groupMsg, ok := mc.GetGroupMessage(); ok {
		return groupMsg.Elements, true
	}
	if tempMsg, ok := mc.GetTempMessage(); ok {
		return tempMsg.Elements, true
	}
	return nil, false
}

func (mc *MessageContext) GetSender() (*message.Sender, bool) {
	if private, ok := mc.GetPrivateMessage(); ok {
		return private.Sender, true
	}
	if groupMsg, ok := mc.GetGroupMessage(); ok {
		return groupMsg.Sender, true
	}
	if tempMsg, ok := mc.GetTempMessage(); ok {
		return tempMsg.Sender, true
	}
	return nil, false
}

func (mc *MessageContext) AssertGroupAndRejectExpired() (string, bool) {
	msg := mc.AssertGroupMessage()

	session, ok := tools.Login.Get(msg.Sender.Uin)
	if !ok {
		mc.SendMessage([]message.IMessageElement{message.NewText("请先使用/login命令登录课程平台")})
		return session, false
	}

	if !tools.CheckSession.CheckSession(session) {
		mc.SendMessage([]message.IMessageElement{message.NewText("登录数据已过期，请重新登录")})
		return session, false
	}
	return session, true
}

func (mc *MessageContext) SendMessage(elements []message.IMessageElement) (interface{}, error) {
	return mc.Message.SendMessage(mc.Client, elements)
}

func (mc *MessageContext) SendFileLocal(localFilePath, filename string, folderId ...string) error {
	return mc.Message.SendFileLocal(mc.Client, localFilePath, filename, folderId...)
}

func (mc *MessageContext) SendFileURL(url, filename string, client *resty.Client, folderId ...string) error {
	return mc.Message.SendFileURL(mc.Client, url, filename, client, folderId...)
}

func (mc *MessageContext) GetMessage() interface{} {
	return mc.Message.GetMessage()
}

// GetContext 获取上下文
func (mc *MessageContext) GetContext() context.Context {
	return mc.ctx
}

// WithContext 设置上下文
func (mc *MessageContext) WithContext(ctx context.Context) *MessageContext {
	mc.ctx = ctx
	return mc
}

// Set 设置元数据
func (mc *MessageContext) Set(key string, value interface{}) {
	mc.Metadata[key] = value
}

// Get 获取元数据
func (mc *MessageContext) Get(key string) (interface{}, bool) {
	value, exists := mc.Metadata[key]
	return value, exists
}

// GetString 获取字符串类型元数据
func (mc *MessageContext) GetString(key string) string {
	if value, exists := mc.Metadata[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetPrivateMessage 获取私聊消息
func (mc *MessageContext) GetPrivateMessage() (*message.PrivateMessage, bool) {
	if msg, ok := mc.Message.GetMessage().(*message.PrivateMessage); ok {
		return msg, true
	}
	return nil, false
}

// GetGroupMessage 获取群消息
func (mc *MessageContext) GetGroupMessage() (*message.GroupMessage, bool) {
	if msg, ok := mc.Message.GetMessage().(*message.GroupMessage); ok {
		return msg, true
	}
	return nil, false
}

// GetTempMessage 获取临时消息
func (mc *MessageContext) GetTempMessage() (*message.TempMessage, bool) {
	if msg, ok := mc.Message.GetMessage().(*message.TempMessage); ok {
		return msg, true
	}
	return nil, false
}

// AssertPrivateMessage 确认已经是私聊消息后调用
func (mc *MessageContext) AssertPrivateMessage() *message.PrivateMessage {
	if msg, ok := mc.GetPrivateMessage(); ok {
		return msg
	}
	panic("Assert Message is Group Message Failed")
}

// AssertGroupMessage 确认已经是群聊消息后调用
func (mc *MessageContext) AssertGroupMessage() *message.GroupMessage {
	if msg, ok := mc.GetGroupMessage(); ok {
		return msg
	}
	panic("Assert Message is Group Message Failed")
}

// AssertTempMessage 确认已经是临时消息后调用
func (mc *MessageContext) AssertTempMessage() *message.TempMessage {
	if msg, ok := mc.GetTempMessage(); ok {
		return msg
	}
	panic("Assert Message is Group Message Failed")
}

func (mc *MessageContext) RejectNotGroupMessage() bool {
	_, ok := mc.GetGroupMessage()
	if !ok {
		mc.SendMessage([]message.IMessageElement{
			message.NewText("请在群聊中使用本指令"),
		})
	}
	return ok
}

// GetMessageText 获取消息文本内容
func (mc *MessageContext) GetText() string {
	return mc.text
}

func (mc *MessageContext) CreateGroupFileFolder(name string) (string, error) {
	grpMsg := mc.AssertGroupMessage()
	err := mc.Client.CreateGroupFolder(grpMsg.GroupUin, "/", name)
	if err != nil {
		return "", err
	}
	_, folders, err := mc.Client.ListGroupRootFiles(grpMsg.GroupUin)
	if err != nil {
		return "", err
	}

	for _, folder := range folders {
		if folder.FolderName == name {
			return folder.FolderID, nil
		}
	}
	return "", errors.New("未找到文件夹")
}

// extractTextFromElements 从消息元素中提取文本
func extractTextFromElements(elements []message.IMessageElement) string {
	var textParts []string
	for _, element := range elements {
		if textElement, ok := element.(*message.TextElement); ok {
			textParts = append(textParts, textElement.Content)
		}
	}
	return strings.Join(textParts, "")
}

// HandlerFunc 处理器函数类型
type HandlerFunc func(ctx *MessageContext) error

// Middleware 中间件类型
type Middleware func(HandlerFunc) HandlerFunc

// Handler 处理器接口
type Handler interface {
	Handle(ctx *MessageContext) error
}

// HandlerAdapter 处理器适配器
type HandlerAdapter struct {
	handler HandlerFunc
}

// NewHandlerAdapter 创建处理器适配器
func NewHandlerAdapter(handler HandlerFunc) *HandlerAdapter {
	return &HandlerAdapter{handler: handler}
}

// Handle 实现Handler接口
func (ha *HandlerAdapter) Handle(ctx *MessageContext) error {
	return ha.handler(ctx)
}

// Route 路由结构
type Route struct {
	Name        string
	Pattern     string
	Handler     Handler
	Middlewares []Middleware
	Matchers    []Matcher
}

// NewRoute 创建新路由
func NewRoute(name string, handler Handler) *Route {
	return &Route{
		Name:        name,
		Handler:     handler,
		Middlewares: make([]Middleware, 0),
		Matchers:    make([]Matcher, 0),
	}
}

// Use 添加中间件
func (r *Route) Use(middleware Middleware) *Route {
	r.Middlewares = append(r.Middlewares, middleware)
	return r
}

// Match 添加匹配器
func (r *Route) Match(matcher Matcher) *Route {
	r.Matchers = append(r.Matchers, matcher)
	return r
}

// Router 路由器
type Router struct {
	routes       []*Route
	middlewares  []Middleware
	errorHandler func(error, *MessageContext)
	mu           sync.RWMutex
}

// NewRouter 创建新路由器
func NewRouter() *Router {
	return &Router{
		routes:      make([]*Route, 0),
		middlewares: make([]Middleware, 0),
		errorHandler: func(err error, ctx *MessageContext) {
			logrus.Errorf("处理消息时发生错误: %v", err)
		},
	}
}

// Use 添加全局中间件
func (router *Router) Use(middleware Middleware) *Router {
	router.mu.Lock()
	defer router.mu.Unlock()
	router.middlewares = append(router.middlewares, middleware)
	return router
}

// AddRoute 添加路由
func (router *Router) AddRoute(route *Route) *Router {
	router.mu.Lock()
	defer router.mu.Unlock()
	router.routes = append(router.routes, route)
	return router
}

// Handle 处理消息
func (router *Router) Handle(ctx *MessageContext) {
	router.mu.RLock()
	routes := make([]*Route, len(router.routes))
	copy(routes, router.routes)
	middlewares := make([]Middleware, len(router.middlewares))
	copy(middlewares, router.middlewares)
	router.mu.RUnlock()

	// 为每个路由执行处理
	for _, route := range routes {
		// 创建完整的中间件链（全局中间件 + 路由中间件）
		handler := route.Handler.Handle

		// 先添加路由中间件
		for i := len(route.Middlewares) - 1; i >= 0; i-- {
			handler = route.Middlewares[i](handler)
		}

		// 再添加全局中间件
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}

		// 检查路由匹配
		matched := true
		for _, matcher := range route.Matchers {
			if !matcher.Match(ctx) {
				matched = false
				break
			}
		}

		if matched {
			if err := handler(ctx); err != nil {
				router.errorHandler(err, ctx)
			}
		}
	}
}
