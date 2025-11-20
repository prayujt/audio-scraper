# audio-scraper

A self-hosted audio scraper written in Go.
It exposes a small HTTP API for searching music and downloading tagged audio files.
Designed to be used from automation tools such as **Apple Shortcuts**.

---

## How It Works

1. The API receives a search request and queries Spotify using client-credentials auth.
2. Results can be passed to `/download`, which:
   - Builds a `DownloadJob` containing all metadata
   - Pushes the job into a **goroutine worker pool**
3. Workers (configured by `WORKER_SIZE`) run in the background:
   - Download audio using **yt-dlp**
   - Write **ID3 metadata**, including cover art
   - Save the final file into `MUSIC_HOME`

This keeps the API fast and responsive while downloads happen asynchronously.
---

## Environment Variables

| Variable | Description |
|---------|-------------|
| **API_PORT** | Port the HTTP server listens on (e.g. `8080`). |
| **SPOTIFY_CLIENT_ID** | Spotify API client ID. |
| **SPOTIFY_CLIENT_SECRET** | Spotify API client secret. |
| **WORKER_SIZE** | Number of worker goroutines processing download jobs. (optional, defaults to 5) |
| **MUSIC_HOME** | Directory where music files are saved (**no trailing slash**). |

Example:

```bash
export API_PORT=8080
export SPOTIFY_CLIENT_ID=your_id
export SPOTIFY_CLIENT_SECRET=your_secret
export WORKER_SIZE=5
export MUSIC_HOME=/music
```

## Running

### Build
```bash
make build
```

### Start the server
```bash
make start
```

## API

### **GET /search**
Searches for music using Spotify metadata.  
Returns a list of matching tracks, including IDs, album info, artist, release date, and thumbnail URL.

### **POST /download**
Accepts one or more selected tracks and queues them for background downloading.  
Each job is placed into a worker queue and processed by a goroutine pool.

---
