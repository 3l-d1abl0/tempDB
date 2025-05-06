
<img src="https://github.com/3l-d1abl0/tempDB/blob/main/assets/temp-db.png"></img>
# TempDB - In-Memory Key-Value Database

TempDB is a simple, lightweight, in-memory key-value store written in Go with persistence capabilities. It provides a Redis-like interface over TCP connections with support for basic data operations and key expiration.

## Features

- **In-Memory Storage**: Fast read and write operations with data stored in memory
- **Segmented Architecture**: Data is distributed across multiple segments for concurrent access
- **Persistence**:
  - Write-Ahead Log (WAL) for durability
  - Periodic snapshots for faster recovery
  - Automatic WAL rotation to manage disk usage
- **Key Expiration**: Set TTL (Time-To-Live) for keys with automatic cleanup
- **Concurrent Access**: Thread-safe operations with fine-grained locking
- **Simple TCP Protocol**: Easy to integrate with any language or system

## Supported Commands

Currently, TempDB supports the following commands:

- `PING` - Test server connectivity
- `GET <key>` - Retrieve a value by key
- `SET <key> value` - Store a key-value pair
- `SET <key> value EX seconds` - Store a key-value pair with expiration time
- `DEL <key>` - Delete a key
- `EXPIRE <key> seconds` - Set expiration time on an existing key
- `FLUSHDB` - Delete all keys from the database
- `TTL <key>` - Get the remaining time-to-live for a key

## Configuration

TempDB uses a YAML configuration file located at `config/config.yaml` by default. You can specify a different configuration file by setting the `CONFIG_PATH` environment variable.

### Configuration Options

```yaml
server:
  port: "8090"  # Server port
  host: "localhost"  # Server host

store:
  segments_per_cpu: 4  # Number of segments per CPU core
  cleanup_interval_seconds: 1  # Interval for checking expired keys
  wal_file_path: wal.log  # Path to the WAL file
  snapshot_file_path: snapshot.db  # Path to the snapshot file
  wal_flush_interval_seconds: 1  # Interval for flushing WAL to disk
  snapshot_interval_seconds: 120  # Interval for creating snapshots
  wal_max_size_bytes: 450  # Maximum size of WAL file before rotation
  wal_max_files: 5  # Maximum number of WAL files to keep
  wal_directory: wal  # Directory for WAL files
```

## Getting Started

### Prerequisites

- Go 1.23 or higher

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/tempDB.git
   cd tempDB
   ```

2. Build the project:
   ```
   go build -o tempdb
   ```

3. Run the server:
   ```
   ./tempdb
   ```

### Connecting to TempDB


Currently, the Database has upgraded the communication protocol to a redis-like protocol.
As of now, the rules are:

1. ```+ Simple String (e.g., +OK\r\n)```
2. ```- Error (e.g., -ERR key not found\r\n)```
3. ```: Integer (e.g., :100\r\n)```
4. ```$ Bulk String (e.g., $6\r\nfoobar\r\n)```
5. ```* Array (e.g., *2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n)```


you can try something like this programatically:
```golang
func main() {
	// Connect to the server (adjust host/port if needed)
	conn, err := net.Dial("tcp", "localhost:8090")
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Example: Send a RESP-formatted PING command
	//command := "*1\r\n$4\r\nPING\r\n"

	//SET
	//command := "*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$7\r\nmyvalue\r\n"
	//command := "*5\r\n$3\r\nSET\r\n$4\r\nRTRT\r\n$7\r\nmyvalue\r\n$2\r\nEX\r\n$3\r\n300\r\n"
	//command := "*3\r\n$3\r\nSET\r\n$3\r\nabc\r\n$3\r\nabc\r\n"

	//GET
	//command := "*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"
	//command := "*2\r\n$3\r\nGET\r\n$3\r\nnnn\r\n"
	command := "*2\r\n$3\r\nGET\r\n$3\r\nabc\r\n"
	//command := "*2\r\n$3\r\nGET\r\n$7\r\ntempkey\r\n"

	//DEL
	//command := "*2\r\n$3\r\nDEL\r\n$5\r\nmykey\r\n"
	//command := "*2\r\n$3\r\nDEL\r\n$3\r\nabc\r\n"
	//command := "*2\r\n$3\r\nDEL\r\n$7\r\ntempkey\r\n"

	//TTl
	//command := "*2\r\n$3\r\nTTL\r\n$4\r\nRTRT\r\n"
	//command := "*2\r\n$3\r\nTTL\r\n$5\r\nmykey\r\n"
	//command := "*2\r\n$3\r\nTTL\r\n$3\r\nnnn\r\n"
	//command := "*2\r\n$3\r\nTTL\r\n$3\r\nabc\r\n"
	//command := "*2\r\n$3\r\nTTL\r\n$7\r\ntempkey\r\n"

	//EXPIRE
	//command := "*3\r\n$6\r\nEXPIRE\r\n$3\r\nabc\r\n$2\r\n20\r\n"

	//FLUSHDB
	//command := "*1\r\n$7\r\nFLUSHDB\r\n"
	fmt.Print("Sending command: ", command)
	_, err = conn.Write([]byte(command))
	if err != nil {
		fmt.Println("Failed to send command:", err)
		return
	}

	// Read and print the response
	reader := bufio.NewReader(conn)
	fmt.Println("Waiting for response...")
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read response:", err)
		return
	}
	fmt.Printf("Response: %s", response)
}

```
OR You can connect to TempDB using any TCP client. For example, using `telnet`:

```
telnet localhost 8090
```
and then issue the commands in the same format.
currently working on  [tempDB-client](https://github.com/3l-d1abl0/tempDB-client) for a smooth communication with the Database.

## Performance Considerations

- The number of segments is determined by the number of CPU cores and the `segments_per_cpu` configuration
- Adjust `cleanup_interval_seconds` based on your expiration needs
- Tune `snapshot_interval_seconds` based on your durability requirements and write load
- Configure `wal_max_size_bytes` and `wal_max_files` based on your disk space constraints


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
