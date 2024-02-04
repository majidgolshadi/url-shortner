package mysql

import "fmt"

// CreateDns creates dsn (Data Source Name) string
func CreateDns(address, dbname, user, pass string, readTimeoutSec int, writeTimeoutSec int) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?parseTime=%s&readTimeout=%ds&writeTimeout=%ds",
		user,
		pass,
		address,
		dbname,
		"true",
		readTimeoutSec,
		writeTimeoutSec)
}
