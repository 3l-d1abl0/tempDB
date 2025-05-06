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
	segments           []*segment
	numSegments        uint32
	persistenceManager *PersistenceManager
}

func NewStore() Store {
	//get db config
	cfg := config.GetStoreConfig()
	numSegments := uint32(runtime.NumCPU() * cfg.SegmentsPerCPU)
	segments := make([]*segment, numSegments)

	//Get a new instance of Persistance Manager
	persistenceManager, err := NewPersistenceManager()
	if err != nil {
		panic(err)
	}

	//Initalize each segment
	for i := range segments {
		segments[i] = &segment{
			mutex:         &sync.RWMutex{},
			kv:            make(map[string]KeyValue),
			cleanupTicker: time.NewTicker(time.Duration(cfg.CleanupIntervalSeconds) * time.Second),
		}

		//goroutine to check expiry for every segment
		go segments[i].cleanupLoop()
	}

	s := Store{
		segments:           segments,
		numSegments:        numSegments,
		persistenceManager: persistenceManager,
	}

	// Load snapshot
	fmt.Println("Reading snapshot...")
	snapshotData, err := persistenceManager.LoadSnapshot()
	if err != nil {
		fmt.Println("Failed to load snapshot:", err) // Log the error, but don't return it
	}

	// Apply snapshot data to segments
	fmt.Println("Applying snapshot...")
	for k, v := range snapshotData {
		segment := s.getSegment(k)
		segment.mutex.Lock()
		segment.kv[k] = v
		segment.mutex.Unlock()
	}

	// Replay WAL
	fmt.Println("Replaying WAL...")
	err = persistenceManager.ReplayWAL(func(record WALRecord) error {
		segment := s.getSegment(record.Key)
		segment.mutex.Lock()
		defer segment.mutex.Unlock()

		switch record.Command {
		case "SET":
			segment.kv[record.Key] = KeyValue{Value: record.Value, ExpireAt: record.ExpireAt}
		case "DEL":
			delete(segment.kv, record.Key)
		case "EXPIRE":
			if _, exists := segment.kv[record.Key]; exists {
				segment.kv[record.Key] = KeyValue{Value: segment.kv[record.Key].Value, ExpireAt: record.ExpireAt}
			}
		case "FLUSHDB":
			for _, seg := range s.segments {
				seg.mutex.Lock()
				defer seg.mutex.Unlock()

				// Clear the map
				seg.kv = make(map[string]KeyValue)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println("Failed to replay WAL:", err) // Log the error, but don't return it
	}

	//Start snapshotting
	s.startSnapshotting()

	return s
}

// Start snapshotting
func (s *Store) startSnapshotting() {
	//Read config
	cfg := config.GetStoreConfig()
	snapshotInterval := time.Duration(cfg.SnapshotIntervalSeconds) * time.Second
	ticker := time.NewTicker(snapshotInterval)

	go func() {
		fmt.Println("Starting snapshotting...")
		for range ticker.C {
			if err := s.createSnapshot(); err != nil {
				fmt.Printf("ERR: Failed to create snapshot: %v\n", err)
			}
		}
	}()
}

func (s *Store) createSnapshot() error {

	//Create a map to hold the data for the snapshot
	snapshotData := make(map[string]KeyValue)

	//Iterate over all segments and copy the data
	for _, segment := range s.segments {
		segment.mutex.RLock()
		for k, v := range segment.kv {
			snapshotData[k] = v
		}
		segment.mutex.RUnlock()
	}

	// Save snapshot
	if err := s.persistenceManager.SaveSnapshot(snapshotData); err != nil {
		return fmt.Errorf("ERR: failed to save snapshot: %w", err)
	}

	// Force WAL rotation after successful snapshot
	if err := s.persistenceManager.RotateWAL(); err != nil {
		return fmt.Errorf("ERR: failed to rotate WAL after snapshot: %w", err)
	}

	fmt.Println("Snapshot and WAL rotation completed successfully")
	return nil
}

func (s *Store) getSegment(key string) *segment {
	//Generate hash for Key
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.segments[h.Sum32()%s.numSegments]
}

func (db *Store) CommandHandler(command utils.Request) ([]byte, error) {

	var record WALRecord

	// Write to WAL
	if command.Command == "FLUSHDB" || command.Command == "PING" { //FLUSHDB and PING commands don't have a key

		record = WALRecord{
			Command:  command.Command,
			Key:      "",       // Assuming first param is always the key
			Value:    []byte{}, // Value will be set in specific command handlers
			ExpireAt: 0,        // ExpireAt will be set in specific command handlers
		}

	} else {

		record = WALRecord{
			Command:  command.Command,
			Key:      command.Params[0], // Assuming first param is always the key
			Value:    []byte{},          // Value will be set in specific command handlers
			ExpireAt: 0,                 // ExpireAt will be set in specific command handlers
		}
	}

	if len(command.Params) > 1 {
		record.Value = []byte(command.Params[1]) // Assuming second param is the value
	}

	if command.Command == "EXPIRE" && len(command.Params) > 1 {
		seconds, err := strconv.ParseInt(command.Params[1], 10, 64)
		if err == nil {
			record.ExpireAt = time.Now().Unix() + seconds
		}
	}

	if db.persistenceManager != nil {
		if err := db.persistenceManager.WriteWALRecord(record); err != nil {
			fmt.Println("Failed to write to WAL:", err) // Log the error, but don't return it
		}
	}

	switch command.Command {
	case "PING":
		return db.Ping()
	case "GET":
		return db.Get(command.Params)
	case "SET":
		return db.Set(command.Params)
	case "DEL":
		return db.Del(command.Params)
	case "FLUSHDB":
		return db.FlushDB()
	case "EXPIRE":
		return db.Expire(command.Params)
	case "TTL":
		return db.TTL(command.Params)
	default:
		return []byte(""), errors.New("invalid command")
	}
}

// Close closes the store and its persistence manager.
func (db *Store) Close() error {
	if db.persistenceManager != nil {
		err := db.persistenceManager.Close()
		if err != nil {
			return err
		}
	}
	return nil
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
