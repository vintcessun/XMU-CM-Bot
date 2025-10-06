package message

import (
	"errors"
	"io"
	"os"

	"github.com/LagrangeDev/LagrangeGo/client"
	"github.com/LagrangeDev/LagrangeGo/message"
	lagio "github.com/LagrangeDev/LagrangeGo/utils/io"
	"github.com/go-resty/resty/v2"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

type Message interface {
	MessageType() string
	GetMessage() interface{}
	SendMessage(client *client.QQClient, elements []message.IMessageElement) (interface{}, error)
	SendFileLocal(client *client.QQClient, localFilePath, filename string, folderId ...string) error
	SendFileURL(client *client.QQClient, url, filename string, restyClient *resty.Client, folderId ...string) error
	GetMessageElements() []message.IMessageElement
	GetSender() *message.Sender
}

type PrivateMessage struct {
	message *message.PrivateMessage
}

func (*PrivateMessage) MessageType() string       { return "私聊消息" }
func (m *PrivateMessage) GetMessage() interface{} { return m.message }
func (m *PrivateMessage) SendMessage(client *client.QQClient, elements []message.IMessageElement) (interface{}, error) {
	return client.SendPrivateMessage(m.message.Sender.Uin, elements)
}
func (m *PrivateMessage) SendFileLocal(client *client.QQClient, localFilePath, filename string, folderId ...string) error {
	return client.SendPrivateFile(m.message.Sender.Uin, localFilePath, filename)
}
func (m *PrivateMessage) SendFileURL(client *client.QQClient, url, filename string, restyClient *resty.Client, folderId ...string) error {
	deferFunc, fileElem, err := getTempFileElementURL(url, filename, restyClient)
	defer deferFunc()
	if err != nil {
		return err
	}

	_, err = client.UploadPrivateFile(m.message.Sender.UID, fileElem)
	if err != nil {
		return err
	}

	return nil
}
func (m *PrivateMessage) GetMessageElements() []message.IMessageElement {
	return m.message.Elements
}
func (m *PrivateMessage) GetSender() *message.Sender {
	return m.message.Sender
}

type TempMessage struct {
	message *message.TempMessage
}

func (*TempMessage) MessageType() string       { return "临时对话消息" }
func (m *TempMessage) GetMessage() interface{} { return m.message }
func (m *TempMessage) SendMessage(client *client.QQClient, elements []message.IMessageElement) (interface{}, error) {
	return client.SendTempMessage(m.message.GroupUin, m.message.Sender.Uin, elements)
}
func (m *TempMessage) SendFileLocal(client *client.QQClient, localFilePath, filename string, folderId ...string) error {
	return errors.New("临时聊天无法发送消息")
}
func (m *TempMessage) SendFileURL(client *client.QQClient, url, filename string, restyClient *resty.Client, folderId ...string) error {
	return errors.New("临时聊天无法发送消息")
}
func (m *TempMessage) GetMessageElements() []message.IMessageElement {
	return m.message.Elements
}
func (m *TempMessage) GetSender() *message.Sender {
	return m.message.Sender
}

type GroupMessage struct {
	message *message.GroupMessage
}

func (*GroupMessage) MessageType() string       { return "群聊消息" }
func (m *GroupMessage) GetMessage() interface{} { return m.message }
func (m *GroupMessage) SendMessage(client *client.QQClient, elements []message.IMessageElement) (interface{}, error) {
	return client.SendGroupMessage(m.message.GroupUin, append([]message.IMessageElement{message.NewAt(m.message.Sender.Uin), message.NewText(" \n")}, elements...))
}
func (m *GroupMessage) SendFileLocal(client *client.QQClient, localFilePath, filename string, folderId ...string) error {
	folder := lagio.Ternary(len(folderId) >= 1 && folderId[0] != "", folderId[0], "/")
	return client.SendGroupFile(m.message.GroupUin, localFilePath, filename, folder)
}
func (m *GroupMessage) SendFileURL(client *client.QQClient, url, filename string, restyClient *resty.Client, folderId ...string) error {
	deferFunc, fileElem, err := getTempFileElementURL(url, filename, restyClient)
	defer deferFunc()
	if err != nil {
		return err
	}

	folder := lagio.Ternary(len(folderId) >= 1 && folderId[0] != "", folderId[0], "/")
	_, err = client.UploadGroupFile(m.message.GroupUin, fileElem, folder)
	if err != nil {
		return err
	}

	return nil
}

func (m *GroupMessage) GetMessageElements() []message.IMessageElement {
	return m.message.Elements
}
func (m *GroupMessage) GetSender() *message.Sender {
	return m.message.Sender
}

func NewMessage(msg any) Message {
	switch v := msg.(type) {
	case *message.PrivateMessage:
		return &PrivateMessage{message: v}
	case message.PrivateMessage:
		return &PrivateMessage{message: &v}
	case *message.TempMessage:
		return &TempMessage{message: v}
	case message.TempMessage:
		return &TempMessage{message: &v}
	case *message.GroupMessage:
		return &GroupMessage{message: v}
	case message.GroupMessage:
		return &GroupMessage{message: &v}
	default:
		panic("消息类型不符合要求")
	}
}

func getTempFileElementURL(url, filename string, client *resty.Client) (func(), *message.FileElement, error) {
	resp, err := client.R().Get(url)
	if err != nil {
		return func() {}, nil, err
	}
	body := resp.RawBody()
	if body != nil {
		defer body.Close()
	}

	tempFile, err := os.CreateTemp("", "send-file-*")
	if err != nil {
		return func() {}, nil, err
	}

	deferFunc := func() {
		if err := tempFile.Close(); err != nil {
			utils.Warn("关闭文件失败")
		}
		os.Remove(tempFile.Name())
	}

	_, err = io.Copy(tempFile, body)
	if err != nil {
		return deferFunc, nil, err
	}

	fileElem := message.NewStreamFile(tempFile, filename)

	return deferFunc, fileElem, nil
}
