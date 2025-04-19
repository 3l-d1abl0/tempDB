package engine

import (
	"errors"
	"fmt"
	"hash/fnv"
	"runtime"
	"strconv"
	"sync"
	"tempDB/config"
	"tempDB/utils"
	"time"
)

type KeyValue struct {
	Value    []byte
	ExpireAt int64 // Unix timestamp for expiration, 0 means no expiration
}

type segment struct {
	mutex         *sync.RWMutex
	kv            map[string]KeyValue
	cleanupTicker *time.Ticker
}

type Store struct {
	segments    []*segment
	numSegments uint32
}

func NewStore() Store {

	//get db config
	cfg := config.GetStoreConfig()
	numSegments := uint32(runtime.NumCPU() * cfg.SegmentsPerCPU)
	segments := make([]*segment, numSegments)

	cleanupInterval := time.Duration(cfg.CleanupIntervalSeconds) * time.Second

	//Initalize each segment
	for i := range segments {
		segments[i] = &segment{
			mutex:         &sync.RWMutex{},
			kv:            make(map[string]KeyValue),
			cleanupTicker: time.NewTicker(cleanupInterval),
		}

		//goroutine to check expiry for every segment
		go segments[i].cleanupLoop()
	}

	return Store{
		segments:    segments,
		numSegments: numSegments,
	}
}

func (s *Store) getSegment(key string) *segment {
	//Generate hash for Key
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.segments[h.Sum32()%s.numSegments]
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

// Handle the parameters for GET command
func (db *Store) executeGet(params []string) ([]byte, error) {

	//KEY
	if len(params) < 1 {
		return []byte(""), errors.New("GET command requires a key")
	}

	key := params[0]
	seg := db.getSegment(key)

	seg.mutex.RLock()
	defer seg.mutex.RUnlock()

	if kv, exists := seg.kv[key]; exists {

		// Check if key has expired
		if kv.ExpireAt != 0 && time.Now().Unix() > kv.ExpireAt {
			//Remove key from db
			delete(seg.kv, key)
			return []byte("(nil)"), nil
		}

		return kv.Value, nil

	}

	return []byte("(nil)"), nil
}

// Handle the parameters for SET command
func (db *Store) executeSet(params []string) ([]byte, error) {

	//KEY VALUE EX 10
	if len(params) < 2 {
		return []byte(""), errors.New("SET command requires key and value")
	}

	key, value := params[0], []byte(params[1])
	fmt.Println(key, value)
	seg := db.getSegment(key)

	fmt.Println("Waiting for Lock !")
	seg.mutex.Lock()
	fmt.Println("Granted")
	defer seg.mutex.Unlock()

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

	seg.kv[key] = KeyValue{
		Value:    value,
		ExpireAt: expireAt,
	}

	return []byte("OK"), nil

}

func (db *Store) executeDel(params []string) ([]byte, error) {
	//KEY
	if len(params) < 1 {
		return []byte(""), errors.New("DEL command requires at least one key")
	}

	key := params[0]
	seg := db.getSegment(key)

	seg.mutex.Lock()
	defer seg.mutex.Unlock()

	if _, exists := seg.kv[key]; exists {
		delete(seg.kv, key)
		return []byte("1"), nil // Returns 1 if key was deleted
	}

	return []byte("0"), nil // Returns 0 if key didn't exist
}

func (db *Store) executeFlushDB() ([]byte, error) {

	//lock all segments for a complete flush
	for _, seg := range db.segments {
		seg.mutex.Lock()
		defer seg.mutex.Unlock()

		// Clear the map
		seg.kv = make(map[string]KeyValue)
	}

	return []byte("OK"), nil
}

func (db *Store) executeExpire(params []string) ([]byte, error) {

	//abc 10
	if len(params) < 2 {
		return []byte(""), errors.New("EXPIRE command requires key and seconds")
	}

	key := params[0]
	seg := db.getSegment(key)

	//base10, should fit in int64
	seconds, err := strconv.ParseInt(params[1], 10, 64)
	if err != nil {
		return []byte(""), errors.New("invalid expire time")
	}

	seg.mutex.Lock()
	defer seg.mutex.Unlock()

	if value, exists := seg.kv[key]; exists {
		value.ExpireAt = time.Now().Unix() + seconds
		seg.kv[key] = value
		return []byte("1"), nil
	}

	return []byte("0"), nil
}

// Cleanup per Segment
func (seg *segment) cleanupLoop() {

	for range seg.cleanupTicker.C {
		seg.mutex.Lock()
		now := time.Now().Unix()
		for k, v := range seg.kv {
			if v.ExpireAt != 0 && now > v.ExpireAt {
				delete(seg.kv, k)
			}
		}
		seg.mutex.Unlock()
	}
}
