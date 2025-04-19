package engine

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"tempDB/utils"
	"time"
)

type KeyValue struct {
	Value    []byte
	ExpireAt int64 // Unix timestamp for expiration, 0 means no expiration
}

type Store struct {
	Mutex         *sync.RWMutex
	Kv            map[string]KeyValue
	cleanupTicker *time.Ticker
}

func NewStore() Store {

	newKeyValueStore := Store{
		Mutex:         &sync.RWMutex{},
		Kv:            make(map[string]KeyValue),
		cleanupTicker: time.NewTicker(time.Second * 1),
	}

	//goroutine to check expiry for entire Database
	go newKeyValueStore.cleanupLoop()
	return newKeyValueStore
}

func (db *Store) CommandHandler(command utils.Request) ([]byte, error) {

	fmt.Println("COMM: ", command.Command)
	fmt.Println("ARGS: ", command.Params)
	switch command.Command {

	case "PING":
		return db.executePing()
	case "GET":
		return db.executeGet(command.Params)
	case "SET":
		return db.executeSet(command.Params)
	case "DEL":
		return db.executeDel(command.Params)
	case "FLUSHDB":
		return db.executeFlushDB()
	case "EXPIRE":
		return db.executeExpire(command.Params)
	default:
		return []byte(""), errors.New("invalid command [Handler]")
	}
}

func (db *Store) executePing() ([]byte, error) {
	return []byte("PONG"), nil
}

func (db *Store) executeGet(params []string) ([]byte, error) {

	if len(params) < 1 {
		return []byte(""), errors.New("GET command requires a key")
	}

	db.Mutex.RLock()
	defer db.Mutex.RUnlock()

	if kv, exists := db.Kv[params[0]]; exists {

		// Check if key has expired
		if kv.ExpireAt != 0 && time.Now().Unix() > kv.ExpireAt {
			//Remove key from db
			delete(db.Kv, params[0])
			return []byte("(nil)"), nil
		}

		return kv.Value, nil

	}

	return []byte("(nil)"), nil
}

func (db *Store) executeSet(params []string) ([]byte, error) {

	if len(params) < 2 {
		return []byte(""), errors.New("SET command requires key and value")
	}

	fmt.Println("Waiting for Lock !")
	db.Mutex.Lock()
	fmt.Println("Granted")
	defer db.Mutex.Unlock()

	key, value := params[0], []byte(params[1])
	fmt.Println(key, value)

	var expireAt int64 = 0
	//Check for Expiry
	if len(params) > 3 && params[2] == "EX" {
		//base10, should fit in int64
		seconds, err := strconv.ParseInt(params[3], 10, 64)
		if err != nil {
			return []byte(""), errors.New("invalid expire time")
		}
		expireAt = time.Now().Unix() + seconds
	}

	db.Kv[key] = KeyValue{
		Value:    value,
		ExpireAt: expireAt,
	}
	return []byte("OK"), nil

}

func (db *Store) executeDel(params []string) ([]byte, error) {
	if len(params) < 1 {
		return []byte(""), errors.New("DEL command requires at least one key")
	}

	db.Mutex.Lock()
	defer db.Mutex.Unlock()

	key := params[0]
	if _, exists := db.Kv[key]; exists {
		delete(db.Kv, key)
		return []byte("1"), nil // Returns 1 if key was deleted
	}

	return []byte("0"), nil // Returns 0 if key didn't exist
}

func (db *Store) executeFlushDB() ([]byte, error) {
	db.Mutex.Lock()
	defer db.Mutex.Unlock()

	// Clear the map
	db.Kv = make(map[string]KeyValue)

	return []byte("OK"), nil
}

func (db *Store) executeExpire(params []string) ([]byte, error) {

	//EXPIRE abc 10
	//abc 10
	if len(params) < 2 {
		return []byte(""), errors.New("EXPIRE command requires key and seconds")
	}

	db.Mutex.Lock()
	defer db.Mutex.Unlock()

	key := params[0]
	//base10, should fit in int64
	seconds, err := strconv.ParseInt(params[1], 10, 64)
	if err != nil {
		return []byte(""), errors.New("invalid expire time")
	}

	if value, exists := db.Kv[key]; exists {
		value.ExpireAt = time.Now().Unix() + seconds
		db.Kv[key] = value
		return []byte("1"), nil
	}

	return []byte("0"), nil
}

func (db *Store) cleanupLoop() {
	for range db.cleanupTicker.C {
		db.Mutex.Lock()
		now := time.Now().Unix()
		for k, v := range db.Kv {
			if v.ExpireAt != 0 && now > v.ExpireAt {
				delete(db.Kv, k)
			}
		}
		db.Mutex.Unlock()
	}
}
