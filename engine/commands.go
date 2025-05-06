package engine

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Handles the parameters for PING command
func (db *Store) Ping() ([]byte, error) {
	return []byte("+PONG\r\n"), nil
}

// Handles the parameters for GET command
func (db *Store) Get(params []string) ([]byte, error) {
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
			return []byte(fmt.Sprintf("+%s\r\n", "(nil)")), nil
		}

		return []byte(fmt.Sprintf("+%s\r\n", kv.Value)), nil
	}

	return []byte(fmt.Sprintf("+%s\r\n", "(nil)")), nil
}

// Handles the parameters for SET command
func (db *Store) Set(params []string) ([]byte, error) {
	//KEY VALUE EX 10
	if len(params) < 2 {
		return []byte(""), errors.New("-ERR SET command requires key and value\r\n")
	}

	key, value := params[0], []byte(params[1])
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
			return []byte(""), errors.New("-ERR invalid expire time\r\n")
		}
		expireAt = time.Now().Unix() + seconds
	}

	seg.kv[key] = KeyValue{
		Value:    value,
		ExpireAt: expireAt,
	}

	return []byte("+OK\r\n"), nil
}

// Handles the parameters for DEL command
func (db *Store) Del(params []string) ([]byte, error) {
	//KEY
	if len(params) < 1 {
		return []byte(""), errors.New("-ERR DEL command requires at least one key\r\n")
	}

	key := params[0]
	seg := db.getSegment(key)

	seg.mutex.Lock()
	defer seg.mutex.Unlock()

	if _, exists := seg.kv[key]; exists {
		delete(seg.kv, key)
		return []byte(":1\r\n"), nil // Returns 1 if key was deleted
	}

	return []byte(":0\r\n"), nil // Returns 0 if key didn't exist
}

// Handles the parameters for TTL command
func (db *Store) TTL(params []string) ([]byte, error) {
	//KEY
	if len(params) < 1 {
		return []byte(""), errors.New("-ERR TTL command requires a key\r\n")
	}

	key := params[0]
	seg := db.getSegment(key)

	seg.mutex.RLock()
	defer seg.mutex.RUnlock()

	if kv, exists := seg.kv[key]; exists {

		//check if no expiration is set
		if kv.ExpireAt == 0 {
			return []byte(":-1\r\n"), nil
		} else if kv.ExpireAt != 0 && time.Now().Unix() > kv.ExpireAt { // Check if key has expired
			//Remove key from db
			delete(seg.kv, key)
			return []byte(":-2\r\n"), nil
		}

		// Calculate TTL
		ttl := kv.ExpireAt - time.Now().Unix()
		return []byte(fmt.Sprintf(":%d\r\n", ttl)), nil
	}
	//does not exist
	return []byte(":-2\r\n"), nil
}

// Handles the parameters for EXPIRE command
func (db *Store) Expire(params []string) ([]byte, error) {
	//abc 10
	if len(params) < 2 {
		return []byte(""), errors.New("-ERR EXPIRE command requires key and seconds")
	}

	key := params[0]
	seg := db.getSegment(key)

	//base10, should fit in int64
	seconds, err := strconv.ParseInt(params[1], 10, 64)
	if err != nil {
		return []byte(""), errors.New("-ERR invalid expire time")
	}

	seg.mutex.Lock()
	defer seg.mutex.Unlock()

	if value, exists := seg.kv[key]; exists {
		value.ExpireAt = time.Now().Unix() + seconds
		seg.kv[key] = value
		return []byte(":1\r\n"), nil
	}

	//key does not exist
	return []byte(":0\r\n"), nil
}

// Handles the parameters for FLUSHDB command
func (db *Store) FlushDB() ([]byte, error) {
	//lock all segments for a complete flush
	for _, seg := range db.segments {
		seg.mutex.Lock()
		defer seg.mutex.Unlock()

		// Clear the map
		seg.kv = make(map[string]KeyValue)
	}

	return []byte("+OK\r\n"), nil
}
