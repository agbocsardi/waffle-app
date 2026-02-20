# Wednesday Waffle - Technical Specifications

## Overview
A lightweight webapp for hosting weekly video check-ins between friends. Users authenticate via invite codes, upload videos, and view videos from their conversation partners. Built with **Golang** and **SQLite** for minimal dependencies.

## Core Functionality

### 1. Authentication
- **Invite Codes**: Users join conversations by entering a reusable invite code (e.g., `waffle-friends-2026`).
- **User Identification**: Users provide a username when joining a conversation. No passwords required.
- **Conversation Creation**: Users can create new conversations and generate an invite code for sharing.

### 2. Conversations
- **Structure**: Each conversation has:
  - A unique `conversation_id` (UUID).
  - An `invite_code` (user-friendly string).
  - A `name` (optional, e.g., "College Friends").
  - A list of `members` (usernames).
- **Access Control**: Only users who have joined a conversation can upload or view its videos.

### 3. Video Handling
- **Upload**: Users upload videos via a web interface or API endpoint. The original video is stored temporarily until transcoding completes.
- **Transcoding**: Videos are automatically transcoded to 720p resolution using `ffmpeg`. If transcoding fails, the original video is retained.
- **Storage**: Videos are stored locally on the server under `/videos/{conversation_id}/`.
- **Metadata**: Each video includes:
  - `uploader` (username).
  - `timestamp` (upload time).
  - `filename` (stored path).

### 4. Video Playback
- **Feed**: Users see a list of videos from their conversation partners, sorted by upload time (newest first).
- **Access**: Users can only view videos from conversations they have joined.

## API Endpoints

### Authentication
- `POST /api/auth/join`: Join a conversation using an invite code. Requires `code` and `username`.
- `GET /api/auth/me`: Get the current user's conversations.

### Conversations
- `POST /api/conversations`: Create a new conversation. Returns `invite_code`.
- `GET /api/conversations`: List all conversations the user has joined.

### Videos
- `POST /api/upload`: Upload a video to a conversation. Requires `conversation_id` and video file.
- `GET /api/videos`: List all videos for a conversation. Requires `conversation_id`.

## Technical Implementation

### Dependencies
- **Language**: Go (Golang).
- **Web Framework**: Standard `net/http` (no external dependencies).
- **Database**: SQLite via `modernc.org/sqlite` (pure Go driver).
- **Transcoding**: System `ffmpeg` (called via `os/exec`).

### Database Schema
- **SQLite**: Used for storing conversation metadata and video metadata.
- **Tables**:
  
  #### Conversations Table
  ```sql
  CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    invite_code TEXT UNIQUE NOT NULL,
    name TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  );
  ```
  
  #### Members Table
  ```sql
  CREATE TABLE members (
    conversation_id TEXT,
    username TEXT,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (conversation_id, username),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
  );
  ```
  
  #### Videos Table
  ```sql
  CREATE TABLE videos (
    id TEXT PRIMARY KEY,
    conversation_id TEXT,
    uploader TEXT,
    filename TEXT,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
  );
  ```

### File Storage
- Videos are stored locally on the filesystem under `/videos/{conversation_id}/`.
- Filenames follow the pattern: `{username}_{timestamp}.mp4`.
- Original videos are stored temporarily until transcoding completes.

### Video Transcoding
- **Input**: Accepts common video formats (MP4, MOV).
- **Output**: Transcodes to 720p MP4 using `ffmpeg`:
  ```bash
  ffmpeg -i input.mp4 -vf "scale=-2:720" output.mp4
  ```
- **Error Handling**: If transcoding fails, the original video is kept, and an error is logged.

### Error Handling
- **Transcoding Failures**: Original videos are retained if `ffmpeg` fails.
- **Database Errors**: Return `500 Internal Server Error` for critical failures.
- **Invalid Requests**: Return `400 Bad Request` for malformed input.

## Security
- **Invite Codes**: No expiration; reusable.
- **Access Control**: Users can only access conversations they have joined.
- **Data Privacy**: Videos are stored locally and not shared outside the conversation.

## Testing
- **Manual Testing**: Use `curl` or Postman to test endpoints.
- **Unit Testing (Optional)**: Use Go's built-in `testing` package for critical functions (e.g., database queries).

## Future Considerations
- Video deletion (manual or automatic).
- Notifications for new videos.
- Mobile app integration.
