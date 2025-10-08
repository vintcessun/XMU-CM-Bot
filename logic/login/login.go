package login

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/LagrangeDev/LagrangeGo/message"
	"github.com/go-resty/resty/v2"
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/tools"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

var qrRegex = regexp.MustCompile(`<input[^>]*?name="execution"[^>]*?value="([^"]*)"[^>]*?>`)
var lntURL = "https://lnt.xmu.edu.cn/"
var retryLoginTimes = 10
var lntURLParsed *url.URL

func showCode(qrcodeData []byte, ctx *event.MessageContext) error {
	_, err := ctx.SendMessage([]message.IMessageElement{
		message.NewText("请使用以下二维码登录："),
		message.NewImage(qrcodeData)})
	return err
}

func init() {
	url, err := url.Parse(lntURL)
	if err != nil {
		panic("login初始化登录地址失败")
	}
	lntURLParsed = url
}

func QrLogin(ctx *event.MessageContext) (string, error) {
	utils.Info("开始二维码登录")
	var session string

	ua := utils.GetFakeUAComputer()
	client := resty.New()
	client.SetHeader("User-Agent", ua)
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(100))

	utils.Trace("二维码登录初始化完毕")

	var retErr error

	for range retryLoginTimes {
		time.Sleep(1 * time.Second)
		utils.Trace("获取登录界面")
		resp, err := client.R().Get(lntURL)
		utils.Trace("登录界面获取完毕")
		if err != nil {
			retErr = err
			utils.Warn("获取登录页面失败")
			continue
		}
		endUrl := resp.RawResponse.Request.URL.String()
		serviceIndex := strings.Index(endUrl, "service=")
		if serviceIndex == -1 {
			utils.Warn("获取服务失败 ", endUrl)
			continue
		}
		service := endUrl[(serviceIndex + 8):]
		utils.Info("获取到服务 service = ", service)
		html := resp.String()

		index := strings.Index(html, "qrLoginForm")
		if index == -1 {
			retErr = err
			utils.Warn("登录execution解析失败")
			continue
		}

		qrHtml := html[index:]

		executions := qrRegex.FindStringSubmatch(qrHtml)

		if len(executions) <= 1 {
			utils.Warn("获得execution解析为空")
			continue
		}

		execution := executions[1]

		utils.Trace("execution = ", execution)

		resp, err = client.R().Get(fmt.Sprintf("https://ids.xmu.edu.cn/authserver/qrCode/getToken?ts=%d", time.Now().UnixMilli()))
		if err != nil {
			retErr = err
			utils.Warn("获得QR标识码失败")
			continue
		}
		qrcodeId := resp.String()

		resp, err = client.R().Get(fmt.Sprintf("https://ids.xmu.edu.cn/authserver/qrCode/getCode?uuid=%s", qrcodeId))
		if err != nil {
			retErr = err
			utils.Warn("获得二维码数据请求失败")
			continue
		}
		body := resp.Body()

		utils.Info("开始展示二维码")
		err = showCode(body, ctx)
		if err != nil {
			retErr = err
			utils.Warn("展示二维码失败")
			continue
		}

		utils.Info("等待二维码请求")

		for {
			resp, err = client.R().Get(fmt.Sprintf("https://ids.xmu.edu.cn/authserver/qrCode/getStatus.htl?ts=%d&uuid=%s", time.Now().UnixMilli(), qrcodeId))
			if err != nil {
				utils.Warn("请求二维码状态请求失败")
				continue
			}

			state := resp.String()
			if state == "0" {
				utils.Trace("等待中")
			} else if state == "1" {
				utils.Info("请求成功")
				break
			} else if state == "2" {
				utils.Trace("已扫描二维码")
			} else if state == "3" {
				utils.Info("二维码已失效")
				return session, errors.New("二维码已失效，登录失败")
			}

			time.Sleep(1 * time.Second)
		}

		data := map[string]string{
			"lt":        "",
			"uuid":      qrcodeId,
			"cllt":      "qrLogin",
			"dllt":      "generalLogin",
			"execution": execution,
			"_eventId":  "submit",
			"rmShown":   "1",
		}

		for range retryLoginTimes {
			resp, err = client.R().SetFormData(data).Post(fmt.Sprintf("https://ids.xmu.edu.cn/authserver/login?display=qrLogin&service=%s", service))
			if err != nil {
				utils.Warn("请求登录请求失败")
				continue
			}

			sessionCookie, ok := utils.GetSessionCookie(client.GetClient().Jar.Cookies(lntURLParsed), "session")
			if !ok {
				utils.Warn("获取cookies失败 cookies = ", client.GetClient().Jar.Cookies(lntURLParsed))
				body := resp.String()
				pureBody := strings.ReplaceAll(strings.ReplaceAll(string(body), " ", ""), "\n", "")
				utils.Warn("body = ", pureBody[:200])
				continue
			}
			session = sessionCookie.Value

			ok = tools.CheckSession.CheckSession(session)
			if !ok {
				utils.Warn("获得的cookie并非最新 session = ", session)
				continue
			}
			break
		}

		break
	}

	if session == "" {
		if retErr == nil {
			retErr = errors.New("系统故障")
		}
		return session, retErr
	} else {
		return session, nil
	}
}

func checkAndLogin(ctx *event.MessageContext) error {
	msg := ctx.AssertGroupMessage()
	session, err := QrLogin(ctx)
	if err != nil {
		ctx.SendMessage([]message.IMessageElement{message.NewText(fmt.Sprintf("登录异常 %v", err))})
		return err
	}
	tools.Login.Insert(msg.Sender.Uin, session)
	ctx.SendMessage([]message.IMessageElement{message.NewText("登录成功，数据已保存")})
	return nil
}

func Login(ctx *event.MessageContext) {
	utils.Info("处理login指令")
	defer utils.Info("处理结束login指令")

	msg := ctx.AssertGroupMessage()
	session, ok := tools.Login.Get(msg.Sender.Uin)

	isSend := false

	if !ok {
		isSend = true
		err := checkAndLogin(ctx)
		if err != nil {
			utils.Warn("登录错误 ", err)
		}
		return
	}

	if !tools.CheckSession.CheckSession(session) {
		ctx.SendMessage([]message.IMessageElement{message.NewText("登录数据已过期，正在重试登录")})
		isSend = true
		err := checkAndLogin(ctx)
		if err != nil {
			utils.Warn("登录错误 ", err)
		}
		return
	}

	if !isSend {
		ctx.SendMessage([]message.IMessageElement{message.NewText("已登录")})
	}
}
