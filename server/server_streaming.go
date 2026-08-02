package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// handleStreamingAPI multiplexes our Netflix-style streaming platform APIs
func (s *Server) handleStreamingAPI(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/api/tmdb/search"):
		s.handleTMDBSearch(w, r)
	case strings.HasPrefix(path, "/api/tmdb/details"):
		s.handleTMDBDetails(w, r)
	case strings.HasPrefix(path, "/api/media/enrich"):
		s.handleMediaEnrich(w, r)
	case strings.HasPrefix(path, "/api/media/list"):
		s.handleMediaList(w, r)
	case strings.HasPrefix(path, "/api/stream"):
		s.handleStream(w, r)
	case strings.HasPrefix(path, "/api/subtitles"):
		s.handleSubtitles(w, r)
	case strings.HasPrefix(path, "/api/progress"):
		s.handleProgress(w, r)
	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// TMDB API search query
func (s *Server) handleTMDBSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		http.Error(w, "TMDB API key is not configured in the server environment", http.StatusInternalServerError)
		return
	}

	// Fetch movies and series from TMDB
	tmdbURL := fmt.Sprintf("https://api.themoviedb.org/3/search/multi?api_key=%s&query=%s", apiKey, url.QueryEscape(query))
	resp, err := http.Get(tmdbURL)
	if err != nil {
		http.Error(w, "Failed to call TMDB API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// TMDB API fetch details
func (s *Server) handleTMDBDetails(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	mediaType := r.URL.Query().Get("type") // 'movie' or 'tv'
	if id == "" || mediaType == "" {
		http.Error(w, "ID and media type are required", http.StatusBadRequest)
		return
	}

	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		http.Error(w, "TMDB API key is not configured", http.StatusInternalServerError)
		return
	}

	tmdbURL := fmt.Sprintf("https://api.themoviedb.org/3/%s/%s?api_key=%s", mediaType, id, apiKey)
	resp, err := http.Get(tmdbURL)
	if err != nil {
		http.Error(w, "Failed to fetch details from TMDB", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// Enrich media file with metadata
func (s *Server) handleMediaEnrich(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath      string  `json:"file_path"`
		TMDBID        int     `json:"tmdb_id"`
		Title         string  `json:"title"`
		MediaType     string  `json:"media_type"`
		Overview      string  `json:"overview"`
		PosterPath    string  `json:"poster_path"`
		BackdropPath  string  `json:"backdrop_path"`
		ReleaseDate   string  `json:"release_date"`
		Genres        string  `json:"genres"`
		Rating        float32 `json:"rating"`
		SeasonNumber  *int    `json:"season_number,omitempty"`
		EpisodeNumber *int    `json:"episode_number,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	if DB == nil {
		http.Error(w, "Database not connected", http.StatusInternalServerError)
		return
	}

	// 1. Check or create MediaItem
	var item MediaItem
	err := DB.Where("tmdb_id = ?", req.TMDBID).First(&item).Error
	if err != nil {
		// Create new MediaItem
		item = MediaItem{
			TMDBID:       req.TMDBID,
			Title:        req.Title,
			MediaType:    req.MediaType,
			Overview:     req.Overview,
			PosterPath:   req.PosterPath,
			BackdropPath: req.BackdropPath,
			ReleaseDate:  req.ReleaseDate,
			Genres:       req.Genres,
			Rating:       req.Rating,
		}
		if err := DB.Create(&item).Error; err != nil {
			http.Error(w, "Failed to create media item: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 2. Associate MediaFile
	var mfile MediaFile
	err = DB.Where("file_path = ?", req.FilePath).First(&mfile).Error
	if err != nil {
		// Create new MediaFile association
		mfile = MediaFile{
			MediaItemID:   item.ID,
			FilePath:      req.FilePath,
			SeasonNumber:  req.SeasonNumber,
			EpisodeNumber: req.EpisodeNumber,
		}
		if err := DB.Create(&mfile).Error; err != nil {
			http.Error(w, "Failed to create media file mapping: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Update existing MediaFile mapping
		mfile.MediaItemID = item.ID
		mfile.SeasonNumber = req.SeasonNumber
		mfile.EpisodeNumber = req.EpisodeNumber
		DB.Save(&mfile)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "media_item_id": item.ID})
}

// Get list of enriched media items for the Netflix grid view
func (s *Server) handleMediaList(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		http.Error(w, "Database not connected", http.StatusInternalServerError)
		return
	}

	var items []MediaItem
	if err := DB.Preload("Files").Find(&items).Error; err != nil {
		http.Error(w, "Failed to retrieve media list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// Handle real-time transcoding with ffmpeg
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	filePathParam := r.URL.Query().Get("path")
	if filePathParam == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	dldir := s.state.Config.DownloadDirectory
	absPath := filepath.Join(dldir, filePathParam)

	// Enforce base path containment security
	if !strings.HasPrefix(absPath, dldir) {
		http.Error(w, "Unauthorized file path access", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found or is a directory", http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	// If the video format is native to standard HTML5 video elements (like MP4), serve directly
	if ext == ".mp4" || ext == ".webm" {
		f, err := os.Open(absPath)
		if err != nil {
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
		return
	}

	// Otherwise, transcode video/audio streams on-the-fly to H.264 / AAC fragmented MP4 container
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Connection", "keep-alive")

	// ffmpeg command for real-time fragmented stream
	cmd := exec.Command("ffmpeg",
		"-i", absPath,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"pipe:1",
	)

	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to execute ffmpeg transcoding: %v\n", err)
		http.Error(w, "Transcoding engine error", http.StatusInternalServerError)
		return
	}

	// Ensure the command is killed if connection terminates abruptly
	notify := r.Context().Done()
	go func() {
		<-notify
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	_ = cmd.Wait()
}

// Convert SRT Subtitles to WebVTT real-time
func (s *Server) handleSubtitles(w http.ResponseWriter, r *http.Request) {
	filePathParam := r.URL.Query().Get("path")
	if filePathParam == "" {
		http.Error(w, "Subtitle file path is required", http.StatusBadRequest)
		return
	}

	dldir := s.state.Config.DownloadDirectory
	absPath := filepath.Join(dldir, filePathParam)

	if !strings.HasPrefix(absPath, dldir) {
		http.Error(w, "Forbidden path mapping", http.StatusForbidden)
		return
	}

	b, err := ioutil.ReadFile(absPath)
	if err != nil {
		http.Error(w, "Failed to load subtitle file", http.StatusNotFound)
		return
	}

	content := string(b)
	ext := strings.ToLower(filepath.Ext(absPath))

	if ext == ".srt" {
		// Convert SRT to WebVTT format
		content = "WEBVTT\n\n" + content

		// Replace commas in timestamps (00:01:20,000) with periods (00:01:20.000)
		re := regexp.MustCompile(`(\d{2}:\d{2}:\d{2}),(\d{3})`)
		content = re.ReplaceAllString(content, "${1}.${2}")
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

// Retrieve or update playback progress in seconds
func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		http.Error(w, "Database not connected", http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		filePathParam := r.URL.Query().Get("path")
		if filePathParam == "" {
			http.Error(w, "Path is required", http.StatusBadRequest)
			return
		}

		var progress PlaybackProgress
		err := DB.Where("file_path = ?", filePathParam).First(&progress).Error
		if err != nil {
			// No progress saved yet
			json.NewEncoder(w).Encode(map[string]interface{}{"progress_seconds": 0, "is_watched": false})
			return
		}

		json.NewEncoder(w).Encode(progress)
		return
	}

	if r.Method == "POST" {
		var req struct {
			FilePath        string `json:"file_path"`
			ProgressSeconds int    `json:"progress_seconds"`
			IsWatched       bool   `json:"is_watched"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid input payload", http.StatusBadRequest)
			return
		}

		var progress PlaybackProgress
		err := DB.Where("file_path = ?", req.FilePath).First(&progress).Error
		if err != nil {
			// Create new record
			progress = PlaybackProgress{
				FilePath:        req.FilePath,
				ProgressSeconds: req.ProgressSeconds,
				IsWatched:       req.IsWatched,
				UpdatedAt:       time.Now(),
			}
			DB.Create(&progress)
		} else {
			// Update existing record
			progress.ProgressSeconds = req.ProgressSeconds
			progress.IsWatched = req.IsWatched
			progress.UpdatedAt = time.Now()
			DB.Save(&progress)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
