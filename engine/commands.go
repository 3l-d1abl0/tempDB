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
