package tools

import (
	"sync"

	"github.com/vintcessun/XMU-CM-Bot/utils"
)

type Department struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Profile struct {
	Department Department `json:"department"`
	Id         int        `json:"id"`
	Name       string     `json:"name"`
	UserNo     string     `json:"user_no"`
}

type ProfileCache struct {
	m sync.Map
}

func (p *ProfileCache) get(key string) (*Profile, bool) {
	data, ok := p.m.Load(key)
	if !ok {
		return nil, ok
	} else {
		switch e := data.(type) {
		case Profile:
			return &e, ok
		case *Profile:
			return e, ok
		default:
			return nil, false
		}
	}
}

func (p *ProfileCache) insert(key string, value *Profile) {
	p.m.Store(key, value)
}

var ProfileCacheValue ProfileCache

func GetProfile(session string) (*Profile, error) {
	profile, ok := ProfileCacheValue.get(session)
	if ok {
		return profile, nil
	}

	client := utils.GetSessionClient(session)
	resp, err := client.R().Get("https://lnt.xmu.edu.cn/api/profile")
	if err != nil {
		return nil, err
	}

	body := resp.Body()
	data, err := utils.UnmarshalJSON[Profile](body)
	if err != nil {
		return nil, err
	}
	ProfileCacheValue.insert(session, data)

	return data, nil
}
