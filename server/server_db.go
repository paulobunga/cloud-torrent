package server

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDB loads dotenv and connects to MySQL with GORM
func ConnectDB() {
	// Attempt to load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, relying on environment variables.")
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Println("Warning: DB_DSN environment variable is empty. MySQL features will be disabled or fail.")
		return
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Error connecting to MySQL database: %v\n", err)
		return
	}

	// Automigrate schemas
	err = db.AutoMigrate(
		&DBSetting{},
		&MediaItem{},
		&MediaFile{},
		&PlaybackProgress{},
	)
	if err != nil {
		log.Printf("Error performing auto-migration: %v\n", err)
		return
	}

	DB = db
	log.Println("Successfully connected to MySQL database and performed auto-migration.")
}
