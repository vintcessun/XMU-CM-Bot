package download

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LagrangeDev/LagrangeGo/message"
	"github.com/go-resty/resty/v2"
	"github.com/vintcessun/XMU-CM-Bot/event"
	"github.com/vintcessun/XMU-CM-Bot/tools"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

var sendFileRetryTime = 3

func intToBase62(num int64) string {
	// 处理0的特殊情况
	if num == 0 {
		return "0"
	}

	// 判断是否为负数
	isNegative := num < 0
	if isNegative {
		num = -num // 取绝对值处理
	}

	// 62进制字符集：0-9、A-Z、a-z
	digits := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var result string

	// 循环计算每一位的62进制表示
	for num > 0 {
		remainder := num % 62
		// 拼接当前位（注意是前置拼接）
		result = string(digits[remainder]) + result
		// 继续处理商
		num = num / 62
	}

	// 如果是负数，添加负号
	if isNegative {
		result = "-" + result
	}

	return result
}

type fileSendErrResponse struct {
	File *tools.FormatFileInside
	Err  error
}

func downloadFunc(session string, command string, ctx *event.MessageContext) ([]message.IMessageElement, error) {
	client := utils.GetSessionClient(session)

	courseData, err := tools.GetCourseData(client)
	if err != nil {
		utils.Warn("获取课程信息失败 ", err)
		return nil, errors.New("获取课程信息失败")
	}

	rencentCourseData, err := tools.GetRecentCourseData(client)
	if err != nil {
		utils.Warn("获取最近课程信息失败 ", err)
		return nil, errors.New("获取最近课程失败")
	}

	course, err := tools.GetLLMChooseCourse(courseData, rencentCourseData, command, client)
	if err != nil {
		return nil, err
	}

	files, err := tools.GetCourseActivities(course.Id, client)
	if err != nil {
		utils.Warn("获取文件失败 ", err)
		return nil, errors.New("获取文件失败")
	}

	folderName := strings.Join([]string{course.Name, "-", intToBase62(time.Now().UnixMilli())}, "")

	folder, err := ctx.CreateGroupFileFolder(folderName)
	if err != nil {
		utils.Warn("创建文件夹失败 ", err)
	}

	errChan := make(chan fileSendErrResponse)
	var wg sync.WaitGroup
	wg.Add(len(*files))

	for _, file := range *files {
		go func() {
			defer wg.Done()
			var err error
			for range sendFileRetryTime {
				err = downloadAndUploadFile(file, folder, client, ctx)
				if err != nil {
					utils.Warn("发送失败，重试 ", err)
					continue
				}
				return
			}
			errChan <- fileSendErrResponse{File: file, Err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	var errData []string

	for errResp := range errChan {
		var errType string
		url, err := tools.GetURLById(errResp.File, client)
		if err != nil {
			errType = "严重错误,可能是系统原因"
		} else {
			errType = fmt.Sprintf("可恢复错误,可能是被腾讯拦截,原文件链接为: %s", url)
		}
		perErr := fmt.Sprintf("失败文件: %s 失败ID: %d 失败原因: %s 失败类型: %s", errResp.File.Name, errResp.File.Id, errResp.Err.Error(), errType)

		errData = append(errData, perErr)

	}

	if len(errData) == 0 {
		return []message.IMessageElement{message.NewText("下载完成")}, nil
	} else {
		return []message.IMessageElement{message.NewText("获取文件失败如下: \n"), message.NewText(strings.Join(errData, "\n"))}, nil
	}
}

func downloadAndUploadFile(file *tools.FormatFileInside, folder string, client *resty.Client, ctx *event.MessageContext) error {
	url, err := tools.GetURLById(file, client)
	if err != nil {
		return err
	}

	err = ctx.SendFileURL(url, file.Name, client, folder)
	if err != nil {
		return err
	}

	return nil
}

func Download(ctx *event.MessageContext) {
	utils.Info("处理download指令")
	defer utils.Info("处理结束download指令")

	command := ctx.GetText()

	session, ok := ctx.AssertGroupAndRejectExpired()
	if !ok {
		return
	}

	result, err := downloadFunc(session, command, ctx)
	for range sendFileRetryTime {
		if err == nil {
			break
		}
		result, err = downloadFunc(session, command, ctx)
	}
	if err != nil {
		utils.Error("下载失败: ", err)
		ctx.SendMessage([]message.IMessageElement{message.NewText("下载失败: " + err.Error())})
		return
	}

	ctx.SendMessage(result)
}
