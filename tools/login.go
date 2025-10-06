package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vintcessun/XMU-CM-Bot/config"
)

var Login LoginStruct

type LoginData struct {
	m sync.Map
}

func (l *LoginData) get(key uint32) (string, bool) {
	data, ok := l.m.Load(key)
	if !ok {
		return "", ok
	} else {
		switch e := data.(type) {
		case string:
			return e, ok
		default:
			return "", false
		}
	}
}

func (l *LoginData) insert(key uint32, value string) {
	l.m.Store(key, value)
}

func (l *LoginData) delete(key uint32) {
	l.m.Delete(key)
}

type LoginStruct struct {
	Data      *LoginData
	cacheFile string
	dirty     *bool
	isRunning *bool
}

func (l *LoginStruct) loadData() error {
	filename := l.cacheFile

	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		Logger.Info("本地不存在login缓存文件跳过")
		return nil
	}

	fileContent, err := os.ReadFile(filename)
	if err != nil {
		Logger.Error("读取login缓存文件失败")
		return err
	}

	err = json.Unmarshal(fileContent, &l.Data)
	if err != nil {
		Logger.Error("login缓存文件解析失败，请尝试删除 %s", filename)
		return err
	}

	return nil
}

func (l *LoginStruct) saveData() error {
	jsonData, err := json.MarshalIndent(l.Data, "", " ")
	if err != nil {
		Logger.Error("序列化数据失败")
		return err
	}

	err = os.WriteFile(l.cacheFile, jsonData, 0644)
	if err != nil {
		Logger.Error("写入文件错误 %s", l.cacheFile)
		return err
	}

	return nil
}

func (l *LoginStruct) Get(uin uint32) (string, bool) {
	ret, ok := l.Data.get(uin)
	return ret, ok
}

func (l *LoginStruct) Insert(uin uint32, session string) {
	l.Data.insert(uin, session)
	*l.dirty = true
}

func (l *LoginStruct) Delete(uin uint32) {
	l.Data.delete(uin)
	*l.dirty = true
}

func (l *LoginStruct) runTaskLoop() {
	const rangeDelay = 1 * time.Second

	defer l.saveData()

	for *l.isRunning {
		time.Sleep(rangeDelay)
		if *l.dirty {
			err := l.saveData()
			if err != nil {
				continue
			}

			*l.dirty = false
			Logger.Info("login数据已保存到磁盘")
		}
	}
}

func (l *LoginStruct) Start() error {
	if *l.isRunning {
		Logger.Warning("Pixiv任务正在运行")
		return nil
	}
	err := l.loadData()
	if err != nil {
		return err
	}

	*l.isRunning = true

	go l.runTaskLoop()

	Logger.Info("Login定时任务已启动")

	return nil
}

func (l *LoginStruct) Stop() error {
	if !*l.isRunning {
		Logger.Warning("Login任务已停止")
		return nil
	}

	*l.isRunning = false

	err := l.saveData()
	if err != nil {
		return err
	}

	Logger.Info("Pixiv定时任务已停止")

	return nil
}

func LoginInit(config *config.Config) error {
	dirty := false
	isRunning := false
	Login = LoginStruct{
		Data:      &LoginData{},
		cacheFile: filepath.Join(config.Bot.CachePath, "login.json"),
		dirty:     &dirty,
		isRunning: &isRunning,
	}

	err := Login.Start()
	if err != nil {
		return err
	}

	return nil
}
