package engine

func (db *Store) Ping() ([]byte, error) {
	return []byte("+PONG\r\n"), nil
}
