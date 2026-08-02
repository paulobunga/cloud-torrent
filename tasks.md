# Project Plan / Tasks List: Cloud Torrent to Netflix-style Streaming Platform

This task list is derived from the architectural analysis of Cloud Torrent. We will implement these features incrementally.

## Phase 1: Database Migration (JSON to MySQL)
- [ ] Install go MySQL driver (`github.com/go-sql-driver/mysql`).
- [ ] Implement database initialization and schema migration in Go.
- [ ] Migrate `cloud-torrent.json` settings storage to the MySQL `settings` table.
- [ ] Verify that current server loads, writes, and updates configuration successfully from the database.

## Phase 2: Metadata Enrichment (TMDB Integration)
- [ ] Create Go backend wrapper for TMDB Search and Details APIs.
- [ ] Add backend endpoints `GET /api/tmdb/search` and `POST /api/media/enrich`.
- [ ] Build Admin Curation Interface in AngularJS (`/admin` panel) allowing admins to select files, search TMDB, and save metadata mapping.
- [ ] Verify metadata saving into MySQL tables `media_items` and `media_files`.

## Phase 3: Transcoding Engine (`ffmpeg`) & Subtitles
- [ ] Create dynamic streaming route `/api/stream?path=...` in Go server.
- [ ] Integrate shell calls to `ffmpeg` for on-the-fly transcoding of unsupported formats to MP4/AAC streams.
- [ ] Implement subtitle converter `/api/subtitles?path=...` converting `.srt` to `.vtt` in real-time.
- [ ] Verify video files (native and transcoded) play cleanly along with WebVTT subtitles in the browser.

## Phase 4: Playback Progress (Resume & Watched Status)
- [ ] Create backend endpoints `GET /api/progress` and `POST /api/progress` to retrieve and store progress.
- [ ] Integrate periodic frontend ping in AngularJS player to update progress in the database.
- [ ] Implement UI indicators for "Watched" and seek-on-load "Resume Watching" prompt in the video player.

## Phase 5: Stream Portal Transformation (Netflix-style UI)
- [ ] Redesign `/` (`index.html`) using a modern, dark grid layout with carousels for Categories/Genres.
- [ ] Fetch and display TMDB posters, backdrops, and media overviews instead of the folder tree.
- [ ] Build a sleek details overlay modal showing metadata, trailers, and a full-page custom video player.
- [ ] Add the "Continue Watching" row using the user's progress.
