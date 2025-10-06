package utils

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func GetSessionClient(session string) *resty.Client {
	client := resty.New()
	ua := GetFakeUAComputer()
	client.SetHeader("User-Agent", ua)
	client.SetHeader("Cookie", fmt.Sprintf("session=%s", session))
	return client
}
