package engine

import (
	"errors"
	"fmt"
	"sync"
	"tempDB/utils"
)

type Store struct {
	Mutex *sync.RWMutex
	Kv    map[string][]byte
}

func NewStore() Store {

	return Store{
		Mutex: &sync.RWMutex{},
		Kv:    make(map[string][]byte),
	}
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
	default:
		return []byte(""), errors.New("invalid command [Handler]")
	}
}

func (db *Store) executePing() ([]byte, error) {
	return []byte("PONG"), nil
}

func (db *Store) executeGet(params []string) ([]byte, error) {

	db.Mutex.RLock()
	defer db.Mutex.RUnlock()
	if value, exists := db.Kv[params[0]]; exists {
		return value, nil
	}
	return []byte("(nil)"), nil
}

func (db *Store) executeSet(params []string) ([]byte, error) {

	fmt.Println("Waiting for Lock !")
	db.Mutex.Lock()
	fmt.Println("Granted")
	defer db.Mutex.Unlock()

	key, value := params[0], []byte(params[1])

	fmt.Println(key, value)
	db.Kv[key] = value
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
	db.Kv = make(map[string][]byte)

	return []byte("OK"), nil
}
