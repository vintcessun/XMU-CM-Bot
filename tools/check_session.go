package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/vintcessun/XMU-CM-Bot/utils"
)

type CheckSessionCache struct{ m sync.Map }

var CheckSession CheckSessionCache

type SessionState struct {
	LastUpdate time.Time
	State      bool
}

var checkDelay = 1 * time.Hour

func (c *CheckSessionCache) get(key string) (*SessionState, bool) {
	data, ok := c.m.Load(key)
	if !ok {
		return nil, ok
	} else {
		switch e := data.(type) {
		case SessionState:
			return &e, ok
		case *SessionState:
			return e, ok
		default:
			return nil, false
		}
	}
}

func (c *CheckSessionCache) insert(key string, value *SessionState) {
	c.m.Store(key, value)
}

func (c *CheckSessionCache) CheckSession(session string) bool {
	state, ok := c.get(session)
	if ok && time.Since(state.LastUpdate) < checkDelay {
		return state.State
	}

	ok, err := checkSessionResult(session)
	if err != nil {
		return true
	}

	c.insert(session, &SessionState{LastUpdate: time.Now(), State: ok})

	return ok
}

var rollcallUrl = "https://lnt.xmu.edu.cn/api/radar/rollcalls?api_version=1.1.0"

func checkSessionResult(session string) (bool, error) {
	ua := utils.GetFakeUA()

	req, err := http.NewRequest("GET", rollcallUrl, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Cookie", fmt.Sprintf("session=%s", session))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Warn("读取Body失败 ", rollcallUrl)
		return false, err
	}

	valid := json.Valid(body)

	return valid, nil
}

func CheckSessionInit() error {
	return nil
}
