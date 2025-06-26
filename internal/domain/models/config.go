package models

// DBConfig holds the configuration parameters for connecting to the database.
type DBConfig struct {
	Host     string // Database host (e.g., "localhost" or IP address)
	Port     string // Database port (e.g., "5432" for PostgreSQL)
	User     string // Database username
	Password string // Database password
	DBName   string // Name of the database
}
