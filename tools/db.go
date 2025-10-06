package tools

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/LagrangeDev/LagrangeGo/message"
	"github.com/vintcessun/XMU-CM-Bot/config"
	message2 "github.com/vintcessun/XMU-CM-Bot/message"
	"github.com/vintcessun/XMU-CM-Bot/utils"
	bolt "go.etcd.io/bbolt"
)

var maxMessageWriteRetry = 3
var Db DB

type DB struct {
	db                    *bolt.DB
	groupTaskInsertChan   chan messageInsertTask[message.GroupMessage]
	groupTaskReadChan     chan messageReadTask[message.GroupMessage]
	privateTaskInsertChan chan messageInsertTask[message.PrivateMessage]
	privateTaskReadChan   chan messageReadTask[message.PrivateMessage]
	tempTaskInsertChan    chan messageInsertTask[message.TempMessage]
	tempTaskReadChan      chan messageReadTask[message.TempMessage]
}

type messageTaskInsertResponse = error

type messageInsertTask[T any] struct {
	Msg      *T
	ID       uint32
	Response chan messageTaskInsertResponse
}

type messageTaskReadResponse[T any] struct {
	Msg *T
	err error
}

type messageReadTask[T any] struct {
	ID       uint32
	Response chan messageTaskReadResponse[T]
}

func (db *DB) ReadMessage(ID uint32) (message2.Message, error) {
	workerCount := 3

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan message2.Message)
	errChan := make(chan error, workerCount)
	var wg sync.WaitGroup

	wg.Add(workerCount)
	go func() {
		defer wg.Done()
		res, err := db.ReadGroupMessage(ID)
		if err != nil {
			errChan <- fmt.Errorf("Group: %v", err)
			return
		}
		resultChan <- message2.NewMessage(res)
	}()
	go func() {
		defer wg.Done()
		res, err := db.ReadPrivateMessage(ID)
		if err != nil {
			errChan <- fmt.Errorf("Private: %v", err)
			return
		}
		resultChan <- message2.NewMessage(res)
	}()
	go func() {
		defer wg.Done()
		res, err := db.ReadTempMessage(ID)
		if err != nil {
			errChan <- fmt.Errorf("Temp: %v", err)
			return
		}
		resultChan <- message2.NewMessage(res)
	}()

	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	select {
	case success := <-resultChan:
		cancel()
		return success, nil
	case <-doneChan:
		close(errChan)
		var allErrors []error
		for err := range errChan {
			allErrors = append(allErrors, err)
		}
		return nil, errors.Join(allErrors...)
	}
}

func (db *DB) ReadGroupMessage(ID uint32) (*message.GroupMessage, error) {
	var err error
	for range maxMessageWriteRetry {
		respCh := make(chan messageTaskReadResponse[message.GroupMessage])
		db.groupTaskReadChan <- messageReadTask[message.GroupMessage]{ID: ID, Response: respCh}
		ret := <-respCh
		if ret.err != nil {
			err = ret.err
			Logger.Debug(fmt.Sprintf("消息读取失败，重试 %#v", ID))
			continue
		}
		return ret.Msg, nil
	}
	return nil, err
}

func (db *DB) InsertGroupMessage(msg *message.GroupMessage) {
	go func() {
		for range maxMessageWriteRetry {
			respCh := make(chan messageTaskInsertResponse)
			db.groupTaskInsertChan <- messageInsertTask[message.GroupMessage]{Msg: msg, ID: msg.ID, Response: respCh}
			err := <-respCh
			if err != nil {
				Logger.Debug(fmt.Sprintf("消息写入失败，重试 %#v", *msg))
				continue
			}
			return
		}
		Logger.Error(fmt.Sprintf("消息写入失败 %#v", *msg))
	}()
}

func (db *DB) ReadPrivateMessage(ID uint32) (*message.PrivateMessage, error) {
	var err error
	for range maxMessageWriteRetry {
		respCh := make(chan messageTaskReadResponse[message.PrivateMessage])
		db.privateTaskReadChan <- messageReadTask[message.PrivateMessage]{ID: ID, Response: respCh}
		ret := <-respCh
		if ret.err != nil {
			err = ret.err
			Logger.Debug(fmt.Sprintf("消息读取失败，重试 %#v", ID))
			continue
		}
		return ret.Msg, nil
	}
	return nil, err
}

func (db *DB) InsertPrivateMessage(msg *message.PrivateMessage) {
	go func() {
		for range maxMessageWriteRetry {
			respCh := make(chan messageTaskInsertResponse)
			db.privateTaskInsertChan <- messageInsertTask[message.PrivateMessage]{Msg: msg, ID: msg.ID, Response: respCh}
			err := <-respCh
			if err != nil {
				Logger.Warning(fmt.Sprintf("消息写入失败，重试 %#v", *msg))
				continue
			}
			return
		}
		Logger.Error(fmt.Sprintf("消息写入失败 %#v", *msg))
	}()
}

func (db *DB) ReadTempMessage(ID uint32) (*message.TempMessage, error) {
	var err error
	for range maxMessageWriteRetry {
		respCh := make(chan messageTaskReadResponse[message.TempMessage])
		db.tempTaskReadChan <- messageReadTask[message.TempMessage]{ID: ID, Response: respCh}
		ret := <-respCh
		if ret.err != nil {
			err = ret.err
			Logger.Debug(fmt.Sprintf("消息读取失败，重试 %#v", ID))
			continue
		}
		return ret.Msg, nil
	}
	return nil, err
}

func (db *DB) InsertTempMessage(msg *message.TempMessage) {
	go func() {
		for range maxMessageWriteRetry {
			respCh := make(chan messageTaskInsertResponse)
			db.tempTaskInsertChan <- messageInsertTask[message.TempMessage]{Msg: msg, ID: msg.ID, Response: respCh}
			err := <-respCh
			if err != nil {
				Logger.Warning(fmt.Sprintf("消息写入失败，重试 %#v", *msg))
				continue
			}
			return
		}
		Logger.Error(fmt.Sprintf("消息写入失败 %#v", *msg))
	}()
}

func (db *DB) DeInit() error {
	return db.db.Close()
}

func uint32ToBytes(id uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, id)
	return b
}

func messageEventGet[T message.GroupMessage | message.PrivateMessage | message.TempMessage](db *bolt.DB, bucketName string, task messageReadTask[T]) messageTaskReadResponse[T] {
	var msg *T
	err := db.View(func(tx *bolt.Tx) error {
		var err error
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("不存在桶 %s", bucketName)
		}

		key := uint32ToBytes(task.ID)

		msgBytes := bucket.Get(key)

		msg, err = utils.UnmarshalJSON[T](msgBytes)
		if err != nil {
			return err
		}

		return nil
	})
	return messageTaskReadResponse[T]{Msg: msg, err: err}
}

func messageEventInsert[T message.GroupMessage | message.PrivateMessage | message.TempMessage](db *bolt.DB, bucketName string, task messageInsertTask[T]) messageTaskInsertResponse {
	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		msgBytes, err := utils.MarshalJSONByte[T](task.Msg)
		if err != nil {
			return err
		}

		key := uint32ToBytes(task.ID)

		if err := bucket.Put(key, msgBytes); err != nil {
			return err
		}

		utils.Debug("成功写入消息 ", task.ID, " ", string(msgBytes))

		return nil
	})
}

func startTaskReadChan[T message.GroupMessage | message.PrivateMessage | message.TempMessage](db *bolt.DB, bucket string, taskCh chan messageReadTask[T]) {
	for task := range taskCh {
		task.Response <- messageEventGet[T](db, bucket, task)
	}
}

func startTaskInsertChan[T message.GroupMessage | message.PrivateMessage | message.TempMessage](db *bolt.DB, bucket string, taskCh chan messageInsertTask[T]) {
	for task := range taskCh {
		task.Response <- messageEventInsert(db, bucket, task)
	}
}

func DBInit(config *config.Config) error {
	db, err := bolt.Open(filepath.Join(config.Bot.CachePath, "message.db"), 0600, nil)
	if err != nil {
		return err
	}

	groupInsertCh := make(chan messageInsertTask[message.GroupMessage])
	groupReadCh := make(chan messageReadTask[message.GroupMessage])
	privateInsertCh := make(chan messageInsertTask[message.PrivateMessage])
	privateReadCh := make(chan messageReadTask[message.PrivateMessage])
	tempInsertCh := make(chan messageInsertTask[message.TempMessage])
	tempReadCh := make(chan messageReadTask[message.TempMessage])

	go startTaskInsertChan(db, "group", groupInsertCh)
	go startTaskReadChan(db, "group", groupReadCh)

	go startTaskInsertChan(db, "private", privateInsertCh)
	go startTaskReadChan(db, "private", privateReadCh)

	go startTaskInsertChan(db, "temp", tempInsertCh)
	go startTaskReadChan(db, "temp", tempReadCh)

	Db = DB{db: db, groupTaskInsertChan: groupInsertCh, groupTaskReadChan: groupReadCh, privateTaskInsertChan: privateInsertCh, privateTaskReadChan: privateReadCh, tempTaskInsertChan: tempInsertCh, tempTaskReadChan: tempReadCh}
	return nil
}
