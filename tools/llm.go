package tools

import (
	"context"
	"encoding/base64"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/vintcessun/XMU-CM-Bot/config"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

var retryTimes = 10
var Llm LLM

type LLMStruct struct {
	Client openai.Client
	Model  string
}

type LLM struct {
	Text    *LLMStruct
	Dynamic *LLMStruct
	Choice  *LLMStruct
}

func GetLLMFromData(data *config.LLMData) *LLMStruct {
	return &LLMStruct{
		Client: openai.NewClient(option.WithAPIKey(data.APIKey), option.WithBaseURL(data.BaseUrl)),
		Model:  data.Model,
	}
}

func LLMInit(c *config.Config) error {
	Llm = LLM{
		Text:    GetLLMFromData(&c.LLM.Text),
		Dynamic: GetLLMFromData(&c.LLM.Dynamic),
		Choice:  GetLLMFromData(&c.LLM.Choice),
	}
	return nil
}

var thinkTagPattern = regexp.MustCompile(`(?s)<thinking>.*?</thinking>`)

func RemoveThinkTags(text string) string {
	return thinkTagPattern.ReplaceAllString(text, "")
}

func GetJsonReturn[T any](llm *LLMStruct, data string) (*T, error) {
	var ret *T

	chatCompletion, err := llm.Client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(data),
		},
		Model: llm.Model,
	})
	if err != nil {
		Logger.Warning("获得LLM返回失败")
		return ret, err
	}

	ret_json := chatCompletion.Choices[0].Message.Content

	ret_json = RemoveThinkTags(ret_json)
	ret_json = strings.ReplaceAll(ret_json, "```json", "")
	ret_json = strings.ReplaceAll(ret_json, "```", "")

	ret, err = utils.UnmarshalJSON[T]([]byte(ret_json))
	if err != nil {
		Logger.Warning("LLM返回格式不正确")
		Logger.Info("LLM返回: %s", ret_json)
		return ret, err
	}

	return ret, nil
}

func LoopGetJsonReturn[T any](llm *LLMStruct, data string) *T {
	var ret *T
	var err error

	for range retryTimes {
		ret, err = GetJsonReturn[T](llm, data)
		if err == nil {
			break
		}
		Logger.Warning("LLM请求失败", err)
		time.Sleep(1 * time.Second)
	}

	return ret
}

func DescribeImageByte(image []byte, extraData string) (string, error) {
	base64Str := base64.StdEncoding.EncodeToString(image)
	imageUrl := "data:image/jpeg;base64," + base64Str

	var ret string
	chatCompletion, err := Llm.Dynamic.Client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("请用具体清晰的语言描述出这张图片除了文字之外的其他东西的情况和状况"),
			openai.UserMessage("可以参考用户的一些特别要求: " + extraData),
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: imageUrl, Detail: "auto"}),
			}),
		},
		Model: Llm.Dynamic.Model,
	})
	if err != nil {
		Logger.Warning("获得LLM返回失败")
		return ret, err
	}

	ret = chatCompletion.Choices[0].Message.Content
	return ret, nil
}

func LoopDescribeImageByte(image []byte, extraData string) string {
	var ret string
	var err error

	for range retryTimes {
		ret, err = DescribeImageByte(image, extraData)
		if err == nil {
			break
		}
		Logger.Warning("LLM请求失败", err)
		time.Sleep(1 * time.Second)
	}

	return ret
}

func DescribeImage(imageUrl string, extraData string) (string, error) {
	var ret string
	chatCompletion, err := Llm.Dynamic.Client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("请用具体清晰的语言描述出这张图片除了文字之外的其他东西的情况和状况"),
			openai.UserMessage("可以参考用户的一些特别要求: " + extraData),
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: imageUrl, Detail: "auto"}),
			}),
		},
		Model: Llm.Dynamic.Model,
	})
	if err != nil {
		Logger.Warning("获得LLM返回失败")
		return ret, err
	}

	ret = chatCompletion.Choices[0].Message.Content
	return ret, nil
}

func LoopDescribeImage(imageUrl string, extraData string) string {
	var ret string
	var err error

	for range retryTimes {
		ret, err = DescribeImage(imageUrl, extraData)
		if err == nil {
			break
		}
		Logger.Warning("LLM请求失败", err)
		time.Sleep(1 * time.Second)
	}

	return ret
}

func DescribeVoice(data []byte, format string, extraData string) (string, error) {
	var ret string
	chatCompletion, err := Llm.Dynamic.Client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("请用具体清晰的语言描述出这段语音除了文字之外的其他东西的情况和状况"),
			openai.UserMessage("可以参考用户的一些特别要求: " + extraData),
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.InputAudioContentPart(openai.ChatCompletionContentPartInputAudioInputAudioParam{Data: base64.StdEncoding.EncodeToString(data), Format: format}),
			}),
		},
		Model: Llm.Dynamic.Model,
	})
	if err != nil {
		Logger.Warning("获得LLM返回失败")
		return ret, err
	}

	ret = chatCompletion.Choices[0].Message.Content
	return ret, nil
}

func LoopDescribeVoice(data []byte, format string, extraData string) string {
	var ret string
	var err error

	for range retryTimes {
		ret, err = DescribeVoice(data, format, extraData)
		if err == nil {
			break
		}
		Logger.Warning("LLM请求失败", err)
		time.Sleep(1 * time.Second)
	}

	return ret
}
