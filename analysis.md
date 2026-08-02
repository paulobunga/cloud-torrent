# Technical Architecture Analysis: Netflix-style Streaming Platform with Cloud Torrent

This document presents a detailed architectural analysis of the current Cloud Torrent application and outlines the requirements and design for building a complete, Netflix-style streaming platform.

---

## 1. Executive Summary

The current Cloud Torrent application is designed as a lightweight, single-user remote torrent downloader. It has:
1. **A lightweight Angular-based Admin panel (`/admin`)** that manages torrent downloads and directory layout.
2. **A basic Stream Portal (`/`)** that displays downloaded folders/files in a recursive tree sidebar, allowing users to watch/download raw files.

To transform this into a robust, multi-tenant/multi-device **Netflix-style streaming platform**, we must address structural limitations on both the backend (Go) and frontend (AngularJS):
- **Database/Persistence**: Transitioning from volatile memory/single `cloud-torrent.json` to a **MySQL** database.
- **Metadata Management**: Creating an interactive admin curation interface that queries **TMDB** to enrich torrent media content with high-quality descriptions, categories, posters, and backdrops.
- **On-the-fly Transcoding**: Integrating `ffmpeg` to transparently stream HLS or container-transcoded videos, ensuring files of all formats (including MKV, AVI, etc.) play smoothly on modern browsers.
- **User Progress**: Tracking playback position ("Resume Watching") and watched states.
- **Subtitles & Assets**: Automatically matching, converting (`.srt` to `.vtt`), and serving subtitles, posters, and trailers.

---

## 2. Identified Issues & Technical Gap Analysis

| Feature Area | Current State in Cloud Torrent | Technical Gap / Issues for Netflix-Style Streaming |
| :--- | :--- | :--- |
| **Data Persistence** | Configuration is stored in a JSON file (`cloud-torrent.json`). In-memory locks manage torrent lists. Files are listed on-the-fly via filesystem traversals. | File listing is slow and doesn't support metadata decoration. No user records, progress logs, or TMDB info can be persisted cleanly without a relational database. |
| **Media Enrichment** | File list only shows file name, size, and modification date. No concept of posters, backdrops, actors, overview, or genre. | Netflix relies heavily on a grid/carousel of rich cards (posters, genres, ratings). On-the-fly directory scanning has no connection to metadata. |
| **Video Playback** | Standard HTML5 `<video>` elements pointing to raw `/download/` files. | HTML5 browsers only natively support MP4 (H.264/AAC), WebM (VP8/VP9), or Ogg. Torrent contents are frequently `.mkv` or `.avi` with HEVC or DTS/AC3 audio, which fail to play and crash or fail silent. |
| **Subtitle Support** | No native support for subtitle tracks in Stream Portal. Subtitles must be downloaded manually with the file. | Subtitles are critical for streaming. Web browsers only support WebVTT (`.vtt`) track elements. Torrent downloads frequently provide `.srt` format, which needs parsing and conversion. |
| **User Experience & Progress** | Stream Portal has no user state. If a browser closes, playback position is completely lost. | "Resume Watching" requires the player to periodically ping play progress (e.g. in seconds) and save it to a database, restoring it upon next load. |

---

## 3. Database Schema Design (MySQL)

To migrate away from `cloud-torrent.json` and support new features, we introduce a relational schema in MySQL:

### 3.1. `settings`
Stores core engine configurations:
- `id`: INT AUTO_INCREMENT PRIMARY KEY
- `download_directory`: VARCHAR(512)
- `incoming_port`: INT
- `enable_upload`: TINYINT(1)
- `enable_seeding`: TINYINT(1)
- `tmdb_api_key`: VARCHAR(256)

### 3.2. `media_items`
Stores rich information for movies, shows, and seasons:
- `id`: INT AUTO_INCREMENT PRIMARY KEY
- `tmdb_id`: INT UNIQUE
- `title`: VARCHAR(255)
- `media_type`: VARCHAR(50) (e.g., 'movie', 'tv')
- `overview`: TEXT
- `poster_path`: VARCHAR(512)
- `backdrop_path`: VARCHAR(512)
- `release_date`: VARCHAR(50)
- `genres`: VARCHAR(512) (comma-separated or JSON list)
- `rating`: FLOAT

### 3.3. `media_files`
Binds the physical filesystem path under `/downloads` to a specific enriched `media_item`:
- `id`: INT AUTO_INCREMENT PRIMARY KEY
- `media_item_id`: INT (Foreign Key referencing `media_items.id`)
- `file_path`: VARCHAR(1024) (Relative path within downloads directory)
- `season_number`: INT (NULL for movies)
- `episode_number`: INT (NULL for movies)
- `duration`: INT (Video duration in seconds, populated via ffprobe)

### 3.4. `playback_progress`
Tracks resume positions for users:
- `id`: INT AUTO_INCREMENT PRIMARY KEY
- `file_path`: VARCHAR(1024) (Unique identifier for the file)
- `progress_seconds`: INT
- `is_watched`: TINYINT(1)
- `updated_at`: TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

---

## 4. Architectural Solution Proposals

### 4.1. TMDB Integration Service
1. Provide a backend endpoint `GET /api/tmdb/search?query=xxx` that calls the TMDB Search API.
2. Provide `GET /api/tmdb/details?id=xxx&type=movie|tv` that calls TMDB for full metadata.
3. In `/admin`, create a Curation Modal or Tab. When an admin selects a downloaded movie/show directory/file, they type a title. The frontend queries the TMDB endpoint, shows matching posters, and on selection, writes the enriched information to MySQL tables (`media_items` and `media_files`).

### 4.2. On-the-fly Transcoding via `ffmpeg`
1. When a user streams a file `GET /api/stream?path=...`, the backend probes the file codec.
2. If the codec is natively supported (e.g., MP4/AAC), we redirect/serve it directly (using `http.ServeContent`).
3. If it requires transcoding, the server starts a background `ffmpeg` process to transcode the video on-the-fly (e.g., transcoding audio to AAC or video to H.264, containerized as fragmented MP4 or served as an live stream). Alternatively, we generate an HLS stream (m3u8 playlist with segmented ts files).

### 4.3. Subtitle Engine (SRT to VTT)
1. Add a backend route `/api/subtitles?path=...`.
2. When called, the backend reads the subtitle file (usually `.srt`). It parses and converts it to `.vtt` format in-memory or caches it, then serves it with the `text/vtt` header.
3. The Stream Portal UI scans the directory for files with `.srt` or `.vtt` extensions sharing the video's base name, rendering them as `<track>` tags inside the `<video>` player.

### 4.4. Progress Synchronization
1. The frontend player registers an `onTimeUpdate` event (every 5-10 seconds) that POSTs to `/api/progress` with `{path: ..., position: ...}`.
2. On initial video playback, the frontend calls `GET /api/progress?path=...` to fetch any existing saved position, asking the user "Would you like to resume from X?" or seeking automatically.

---

## 5. Frontend Redesign: Stream Portal (Netflix Grid UI)

The frontend Stream Portal (`index.html`) will be overhauled from a simple sidebar-and-list to a gorgeous, media-centric layout:
- **Hero banner**: Showcases a popular or last-watched movie with high-res backdrop, description, and "Play" button.
- **Media Row Carousels**: Segmented rows (e.g., "Recently Watched", "Movies", "TV Shows", "Recently Added") with horizontal-scrolling posters.
- **Interactive Modals**: Clicking on a card opens a Netflix-like detail pane with large backdrop, overview, episodes (if show), trailer play option, and a big "Watch" button.
- **Embedded Player**: Seamless custom media player with interactive subtitle controls, quality controls, and a "skip intro" or "resume" prompt.
