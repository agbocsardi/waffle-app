# Wednesday Waffle

A lightweight webapp for hosting weekly short video check-ins between friends.

## Prerequisites

- [Go 1.21+](https://go.dev/dl/) (tested with 1.26)
- [FFmpeg](https://ffmpeg.org/) installed and on your `PATH`

```bash
# macOS
brew install go ffmpeg
```

## Running

```bash
go run ./cmd/server
```

Server starts on `http://localhost:8080`.

## Testing

```bash
go test ./...
```

---

## API Reference

### Join a conversation
Creates a session for the user if the invite code is valid.

```bash
POST /api/conversations/join
Content-Type: application/json

{ "invite_code": "abc123", "username": "alice" }
```

Response sets a `waffle_session` cookie used for all subsequent requests.

---

### Create a conversation
Requires authentication. The creator is automatically added as a member.

```bash
POST /api/conversations
Content-Type: application/json

{ "name": "College Friends" }
```

Response:
```json
{ "id": "...", "invite_code": "abc123", "name": "College Friends" }
```

---

### List your conversations

```bash
GET /api/conversations
```

---

### Upload a video

```bash
POST /api/upload
Content-Type: multipart/form-data

fields:
  - file            (video file, supported: .mp4 .mov .avi .mkv)
  - conversation_id (string)
```

Upload is accepted immediately (HTTP 202). Transcoding to 720p MP4 happens in the background with up to 3 retries. Original file is deleted only after successful transcoding.

---

### List videos in a conversation

```bash
GET /api/videos?conversation_id=<id>
```

Response:
```json
[
  {
    "id": "...",
    "uploader": "alice",
    "status": "ready",
    "uploaded_at": "2026-02-20T12:00:00Z"
  }
]
```

Video statuses: `pending`, `ready`, `error`.
