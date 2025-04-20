package engine

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"tempDB/config"
	"time"
)

// WALRecord represents a record in the Write-Ahead Log.
type WALRecord struct {
	Timestamp int64
	Command   string
	Key       string
	Value     []byte
	ExpireAt  int64
}

// PersistenceManager manages the WAL and snapshotting logic.
type PersistenceManager struct {
	walFile         *os.File
	walEncoder      *gob.Encoder
	snapshotFile    *os.File
	snapshotDecoder *gob.Decoder
	mutex           *sync.Mutex
}

// NewPersistenceManager creates a new PersistenceManager.
func NewPersistenceManager() (*PersistenceManager, error) {

	//Register the type
	gob.Register(WALRecord{})

	pm := &PersistenceManager{
		mutex: &sync.Mutex{},
	}
	cfg := config.GetStoreConfig()

	//Open Wal
	walFile, err := os.OpenFile(cfg.WALFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	pm.walFile = walFile
	pm.walEncoder = gob.NewEncoder(walFile)

	//Open snapshot
	snapshotFile, err := os.OpenFile(cfg.SnapshotFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	pm.snapshotFile = snapshotFile
	pm.snapshotDecoder = gob.NewDecoder(snapshotFile)

	return pm, nil
}

// Open opens the WAL and snapshot files.
func (pm *PersistenceManager) Open() error {
	//Already Handled in NewPersistenceManager
	return nil
}

// Close closes the WAL and snapshot files.
func (pm *PersistenceManager) Close() error {
	if pm.walFile != nil {
		if err := pm.walFile.Close(); err != nil {
			return err
		}
	}
	if pm.snapshotFile != nil {
		if err := pm.snapshotFile.Close(); err != nil {
			return err
		}
	}
	return nil
}

// WriteWALRecord writes a record to the WAL.
func (pm *PersistenceManager) WriteWALRecord(record WALRecord) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	record.Timestamp = time.Now().Unix()

	return pm.walEncoder.Encode(record)
}

// LoadSnapshot loads the database from the snapshot file.
func (pm *PersistenceManager) LoadSnapshot() (map[string]KeyValue, error) {
	data := make(map[string]KeyValue)
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Reset the file pointer to the beginning of the file
	_, err := pm.snapshotFile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek snapshot file: %w", err)
	}

	decoder := json.NewDecoder(pm.snapshotFile)
	err = decoder.Decode(&data)
	if err != nil && err.Error() != "EOF" {
		fmt.Println("Error decoding snapshot:", err)
		return make(map[string]KeyValue), nil // Return empty map, but don't return the error
	}

	return data, nil
}

// SaveSnapshot saves the database to the snapshot file.
func (pm *PersistenceManager) SaveSnapshot(data map[string]KeyValue) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Reset the file pointer to the beginning of the file
	_, err := pm.snapshotFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek snapshot file: %w", err)
	}

	// Truncate the file to remove any previous content
	err = os.Truncate(pm.snapshotFile.Name(), 0)
	if err != nil {
		return fmt.Errorf("failed to truncate snapshot file: %w", err)
	}

	encoder := json.NewEncoder(pm.snapshotFile)
	err = encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode snapshot data: %w", err)
	}

	return nil
}

// ReplayWAL replays the WAL log to restore the database.
func (pm *PersistenceManager) ReplayWAL(apply func(record WALRecord) error) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Reset the file pointer to the beginning of the file
	_, err := pm.walFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek WAL file: %w", err)
	}

	reader := bufio.NewReader(pm.walFile)
	decoder := gob.NewDecoder(reader)

	for {
		var record WALRecord
		err := decoder.Decode(&record)
		if err != nil {
			if err.Error() == "EOF" {
				break // End of file
			}
			return fmt.Errorf("failed to decode WAL record: %w", err)
		}

		err = apply(record)
		if err != nil {
			return fmt.Errorf("failed to apply WAL record: %w", err)
		}
	}

	return nil
}
