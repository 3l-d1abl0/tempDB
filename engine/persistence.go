package engine

import (
	"encoding/gob"
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
	return nil, nil
}

// SaveSnapshot saves the database to the snapshot file.
func (pm *PersistenceManager) SaveSnapshot(data map[string]KeyValue) error {
	return nil
}

// ReplayWAL replays the WAL log to restore the database.
func (pm *PersistenceManager) ReplayWAL(apply func(record WALRecord) error) error {
	return nil
}
