package engine

import (
	"errors"
	"fmt"
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
