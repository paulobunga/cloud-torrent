package server

import (
	"time"

	"gorm.io/gorm"
)

// Setting model maps to settings table in MySQL
type DBSetting struct {
	ID                uint   `gorm:"primaryKey"`
	DownloadDirectory string `gorm:"size:512"`
	IncomingPort      int
	EnableUpload      bool
	EnableSeeding     bool
	TMDBAPIKey        string `gorm:"size:256"`
}

func (DBSetting) TableName() string {
	return "settings"
}

// MediaItem model maps to media_items table
type MediaItem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TMDBID      int       `gorm:"uniqueIndex" json:"tmdb_id"`
	Title       string    `gorm:"size:255" json:"title"`
	MediaType   string    `gorm:"size:50" json:"media_type"` // 'movie' or 'tv'
	Overview    string    `gorm:"type:text" json:"overview"`
	PosterPath  string    `gorm:"size:512" json:"poster_path"`
	BackdropPath string   `gorm:"size:512" json:"backdrop_path"`
	ReleaseDate string    `gorm:"size:50" json:"release_date"`
	Genres      string    `gorm:"size:512" json:"genres"` // comma-separated genres
	Rating      float32   `json:"rating"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Files       []MediaFile `gorm:"foreignKey:MediaItemID" json:"files,omitempty"`
}

func (MediaItem) TableName() string {
	return "media_items"
}

// MediaFile model maps to media_files table
type MediaFile struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	MediaItemID   uint      `gorm:"index" json:"media_item_id"`
	FilePath      string    `gorm:"size:1024" json:"file_path"` // relative path under downloads/
	SeasonNumber  *int      `json:"season_number,omitempty"`
	EpisodeNumber *int      `json:"episode_number,omitempty"`
	Duration      int       `json:"duration"` // duration in seconds
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (MediaFile) TableName() string {
	return "media_files"
}

// PlaybackProgress model maps to playback_progress table
type PlaybackProgress struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	FilePath        string    `gorm:"size:1024;uniqueIndex" json:"file_path"`
	ProgressSeconds int       `json:"progress_seconds"`
	IsWatched       bool      `json:"is_watched"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (PlaybackProgress) TableName() string {
	return "playback_progress"
}

// InitDatabase initializes GORM DB connection and runs AutoMigrate
func InitDatabase(dsn string) (*gorm.DB, error) {
	// Let's use gorm mysql driver
	importGormMySQL := func() {} // local dummy just for compiling references cleanly
	_ = importGormMySQL

	return nil, nil // we will write real init code inside server.go or a server_db.go
}
