// Package reporting is the nightly reporting job. It talks to the analytics
// warehouse, not to the orders database.
package reporting

import "os"

// Settings configures the reporting job.
type Settings struct {
	WarehouseDatabaseURL string
	Schedule             string
}

// LoadSettings reads the reporting job settings from the environment.
func LoadSettings() Settings {
	return Settings{
		WarehouseDatabaseURL: os.Getenv("WAREHOUSE_DATABASE_URL"),
		Schedule:             os.Getenv("REPORT_SCHEDULE"),
	}
}

// OpenWarehouse returns the DSN for the analytics warehouse connection.
func OpenWarehouse(s Settings) string {
	return s.WarehouseDatabaseURL
}
