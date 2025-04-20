package engine

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	currentWALFile  string
	walFileMaxSize  int64
	maxWALFiles     int
	walDirectory    string
}

// NewPersistenceManager creates a new PersistenceManager.
func NewPersistenceManager() (*PersistenceManager, error) {
	//Register the type
	gob.Register(WALRecord{})

	cfg := config.GetStoreConfig()
	pm := &PersistenceManager{
		mutex:          &sync.Mutex{},
		currentWALFile: cfg.WALFilePath,
		walFileMaxSize: cfg.WALMaxSizeBytes,
		maxWALFiles:    cfg.WALMaxFiles,
		walDirectory:   cfg.WALDirectory,
	}

	//Create walDirectory if it doesn't exist
	if err := os.MkdirAll(cfg.WALDirectory, 0755); err != nil {
		return nil, err
	}

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
		fmt.Println("Closed wal File")
	}
	if pm.snapshotFile != nil {
		if err := pm.snapshotFile.Close(); err != nil {
			return err
		}
		fmt.Println("Closed Snapshot File")
	}
	return nil
}

// RotateWAL handles WAL file rotation
func (pm *PersistenceManager) RotateWAL() error {

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Get current WAL file info
	fileInfo, err := pm.walFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get WAL file info: %w", err)
	}

	// Check if rotation is needed
	if fileInfo.Size() < pm.walFileMaxSize {
		fmt.Printf("INFO: No Rotation : WAL file size: %d Bytes, max size: %dBytes\n", fileInfo.Size(), pm.walFileMaxSize)
		return nil
	}

	fmt.Println("Preparing for Rotating WAL...")

	// Close current WAL file
	if err := pm.walFile.Close(); err != nil {
		return fmt.Errorf("failed to close current WAL: %w", err)
	}
	fmt.Println("Closed Current wal File")

	// Generate new WAL filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	newWALPath := filepath.Join(pm.walDirectory, fmt.Sprintf("wal-%s.log", timestamp))

	// Rename current WAL to archived name (with timestamp)
	fmt.Println("New Wal", newWALPath)
	if err := os.Rename(pm.currentWALFile, newWALPath); err != nil {
		return fmt.Errorf("failed to rename WAL file: %w", err)
	}

	// Create new WAL file
	walFile, err := os.OpenFile(pm.currentWALFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new WAL file: %w", err)
	}

	pm.walFile = walFile
	pm.walEncoder = gob.NewEncoder(walFile)

	// Cleanup old WAL files
	fmt.Println("Cleaning up old WAL files...")
	if err := pm.cleanupOldWALFiles(); err != nil {
		fmt.Printf("Warning: failed to cleanup old WAL files: %v\n", err)
	}

	return nil
}

// CleanupOldWALFiles removes old WAL files exceeding maxWALFiles
func (pm *PersistenceManager) cleanupOldWALFiles() error {
	files, err := filepath.Glob(filepath.Join(pm.walDirectory, "wal-*.log"))
	if err != nil {
		return err
	}

	// Sort files by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		iInfo, _ := os.Stat(files[i])
		jInfo, _ := os.Stat(files[j])
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	// Remove files exceeding maxWALFiles
	for i := pm.maxWALFiles; i < len(files); i++ {
		if err := os.Remove(files[i]); err != nil {
			return fmt.Errorf("failed to remove old WAL file %s: %w", files[i], err)
		}
	}

	return nil
}

// Modified WriteWALRecord to check for rotation
func (pm *PersistenceManager) WriteWALRecord(record WALRecord) error {
	if err := pm.RotateWAL(); err != nil {
		return fmt.Errorf("failed to rotate WAL: %w", err)
	}

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
