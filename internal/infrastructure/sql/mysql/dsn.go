package mysql

import "fmt"

// CreateDSN creates a DSN (Data Source Name) string for MySQL connections.
func CreateDSN(address, dbname, user, pass string, readTimeoutSec int, writeTimeoutSec int) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?parseTime=true&readTimeout=%ds&writeTimeout=%ds",
		user,
		pass,
		address,
		dbname,
		readTimeoutSec,
		writeTimeoutSec,
	)
}