# shlx (ShellX)

Fast, collaborative live terminal sharing tool built with Go and SvelteKit.

## Features

- **Real-time terminal sharing** - Share your terminal with others instantly
- **Multi-user collaboration** - Multiple users can view and interact with the same terminal
- **Role-based access** - Owner, interactive, and read-only roles
- **Low latency** - WebSocket-based communication for minimal overhead
- **Secure by default** - Session-based access with owner tokens
- **Clean UI** - Minimal dark interface with xterm.js
- **User presence** - See who's connected in real-time

## Architecture

### Backend (Go)
- **PTY Management** - Spawns and manages real pseudo-terminals using `creack/pty`
- **Session Manager** - Handles multiple concurrent sessions with role-based access control
- **WebSocket Server** - Real-time bidirectional communication using `gorilla/websocket`
- **Clean separation** - Modular design with separate packages for pty, session, and websocket handling

### Frontend (SvelteKit)
- **Terminal rendering** - xterm.js with fit and weblinks addons
- **Real-time sync** - WebSocket client for live terminal updates
- **Session management** - Create and join sessions via simple UI
- **User presence** - Live indicators showing connected users and their roles

## Quick Start

### Prerequisites
- Go 1.21 or higher
- Node.js 18 or higher
- npm or pnpm

### Development Setup

1. **Clone and navigate to the project:**
```bash
git clone https://github.com/mqverk/shlx.git
cd shlx
```

2. **Install Go dependencies:**
```bash
go mod download
```

3. **Install frontend dependencies:**
```bash
cd frontend
npm install
cd ..
```

4. **Run in development mode:**

Terminal 1 - Start the Go backend:
```bash
go run main.go
```

Terminal 2 - Start the frontend dev server:
```bash
cd frontend
npm run dev
```

5. **Access the application:**
Open http://localhost:5173 in your browser.

### Production Build

Build everything at once:
```bash
chmod +x build.sh
./build.sh
```

Run the production server:
```bash
./shlx
```

The server will serve both the API and the built frontend on http://localhost:8080.

### Custom Configuration

Run with custom options:
```bash
./shlx -port 3000 -host 0.0.0.0 -shell /bin/zsh -verbose
```

Options:
- `-port` - Server port (default: 8080)
- `-host` - Server host (default: localhost)
- `-shell` - Shell to use (default: $SHELL or /bin/bash)
- `-verbose` - Enable verbose logging

## Usage

### Create a Session

**Via UI:**
1. Go to http://localhost:8080
2. Click "Create New Session"
3. Copy the session ID and owner token
4. Share the session URL with collaborators

**Via API:**
```bash
curl -X POST http://localhost:8080/api/create
```

Response:
```json
{
  "sessionId": "abc12345",
  "ownerToken": "uuid-token-here",
  "url": "http://localhost:8080/session/abc12345"
}
```

### Join a Session

**As owner (full control):**
```
http://localhost:8080/session/{sessionId}?token={ownerToken}
```

**As interactive user (can read and write):**
```
http://localhost:8080/session/{sessionId}?role=interactive
```

**As read-only viewer:**
```
http://localhost:8080/session/{sessionId}?role=readonly
```

### Roles

- **Owner** - Full control, created the session (requires owner token)
- **Interactive** - Can read and write to the terminal
- **Read-only** - Can only view terminal output

## API Reference

### Create Session
```
POST /api/create
```

Creates a new terminal session and returns session details.

### Get Session Info
```
GET /api/session/{sessionId}
```

Returns information about an active session including connected users.

### WebSocket Connection
```
WS /ws
```

WebSocket endpoint for terminal communication. First message must be a join message:
```json
{
  "sessionId": "session-id",
  "token": "owner-token-or-empty",
  "role": "interactive-or-readonly"
}
```

### Health Check
```
GET /health
```

Returns server health status.

## Project Structure

```
shlx/
├── backend/
│   ├── pty/           # PTY management
│   │   └── pty.go
│   ├── session/       # Session and user management
│   │   └── session.go
│   └── websocket/     # WebSocket handlers
│       ├── handler.go
│       └── reader.go
├── frontend/
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +page.svelte              # Home page
│   │   │   ├── +layout.svelte            # Layout
│   │   │   └── session/[id]/
│   │   │       ├── +page.svelte          # Terminal session page
│   │   │       └── +page.js              # Page config
│   │   ├── app.html                       # HTML template
│   │   └── app.css                        # Global styles
│   ├── package.json
│   ├── vite.config.js
│   └── svelte.config.js
├── main.go            # Application entry point
├── go.mod
├── build.sh           # Build script
└── README.md
```

## Security Considerations

**For production use:**

1. **Enable CORS properly** - Update `CheckOrigin` in [backend/websocket/handler.go](backend/websocket/handler.go)
2. **Use HTTPS/WSS** - Set up TLS certificates for encrypted connections
3. **Add authentication** - Implement proper user authentication
4. **Session timeouts** - Add automatic session cleanup
5. **Rate limiting** - Protect against abuse
6. **Input validation** - Validate all user inputs

**Current state:**
This is a functional prototype. The CORS policy is permissive (`CheckOrigin: true`) for development ease. Tighten security before deploying to production.

## Performance

- **Low latency** - Direct WebSocket connections, no polling
- **Minimal overhead** - Efficient binary message passing
- **Concurrent safe** - Proper mutex locking for multi-user access
- **Buffered channels** - Non-blocking broadcast to prevent slow clients from affecting others

## Troubleshooting

**"Session not found" error:**
- Verify the session ID is correct
- Sessions are in-memory only and lost on server restart

**Terminal not rendering:**
- Check browser console for errors
- Ensure WebSocket connection is established
- Verify the backend is running

**Cannot type in terminal:**
- Check your role (read-only users cannot type)
- Ensure WebSocket connection is active

**Build fails:**
- Run `go mod download` to fetch Go dependencies
- Run `npm install` in frontend directory
- Check Go and Node.js versions

## License

MIT License - feel free to use this project for any purpose.

## Contributing

This is a minimal implementation focused on core functionality. Contributions are welcome!

**Potential improvements:**
- Session persistence (Redis/database)
- Terminal recording and playback
- File upload/download
- Copy/paste support improvements
- Mobile responsive terminal
- Collaborative cursor tracking
- Chat/comments system

---

Built with ❤️ using Go and SvelteKit
