# Quick Start Guide

This guide will help you get Alicia running on your machine in minutes. You'll set up the database, start the backend services, launch the agent, and connect with the web frontend.

## Prerequisites

### Recommended: Nix Setup

The easiest way to set up all dependencies is using Nix:

**Install Nix** (if not already installed):
```bash
curl -L https://nixos.org/nix/install | sh
```

**Enable flakes** (add to `~/.config/nix/nix.conf` or `/etc/nix/nix.conf`):
```
experimental-features = nix-command flakes
```

**Enter development shell**:
```bash
cd alicia
nix develop
```

This provides all necessary tools: Go, Node.js, PostgreSQL client, migrate, sqlc, etc.

### Manual Setup

If not using Nix, install these dependencies manually:

1. **Go 1.21+**: [https://go.dev/dl/](https://go.dev/dl/)
2. **Node.js 20+**: [https://nodejs.org/](https://nodejs.org/)
3. **PostgreSQL 15+** with **pgvector** extension
4. **migrate**: Database migration tool ([https://github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate))

### External Services

Alicia requires these external services to be running:

1. **LiveKit Server**: Real-time communication (WebRTC)
   - Install: [https://docs.livekit.io/home/self-hosting/deployment/](https://docs.livekit.io/home/self-hosting/deployment/)
   - Or use LiveKit Cloud: [https://livekit.io/](https://livekit.io/)

2. **LLM Server**: Language model inference (vLLM recommended)
   - Qwen3-8B-AWQ or compatible OpenAI-compatible API
   - Install vLLM: [https://docs.vllm.ai/en/latest/](https://docs.vllm.ai/en/latest/)
   - Example: `vllm serve Qwen/Qwen2.5-7B-Instruct-AWQ --port 8000`

3. **ASR/TTS Server**: Speech recognition and synthesis
   - speaches server (recommended): [https://github.com/longregen/speaches](https://github.com/longregen/speaches)
   - Or OpenAI-compatible ASR/TTS endpoints

## Step-by-Step Setup

### 1. Clone Repository

```bash
git clone https://github.com/longregen/alicia.git
cd alicia
```

### 2. Configure Environment

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```bash
# Database
ALICIA_POSTGRES_URL=postgres://alicia:password@localhost:5432/alicia?sslmode=disable

# LLM (vLLM server)
ALICIA_LLM_URL=http://localhost:8000/v1
ALICIA_LLM_MODEL=Qwen/Qwen2.5-7B-Instruct-AWQ
ALICIA_LLM_API_KEY=sk-dummy  # or actual key if required
ALICIA_LLM_MAX_TOKENS=2048
ALICIA_LLM_TEMPERATURE=0.7

# LiveKit
ALICIA_LIVEKIT_URL=ws://localhost:7880
ALICIA_LIVEKIT_API_KEY=devkey
ALICIA_LIVEKIT_API_SECRET=secret

# ASR (Automatic Speech Recognition)
ALICIA_ASR_URL=http://localhost:8765/v1
ALICIA_ASR_MODEL=openai/whisper-large-v3
ALICIA_ASR_API_KEY=

# TTS (Text-to-Speech)
ALICIA_TTS_URL=http://localhost:8765/v1
ALICIA_TTS_MODEL=hexgrad/kokoro-v0_19
ALICIA_TTS_VOICE=af_sky
ALICIA_TTS_API_KEY=

# Embeddings (for memory search)
ALICIA_EMBEDDING_URL=http://localhost:8000/v1
ALICIA_EMBEDDING_API_KEY=
ALICIA_EMBEDDING_MODEL=Alibaba-NLP/gte-base-en-v1.5

# Server
ALICIA_SERVER_HOST=0.0.0.0
ALICIA_SERVER_PORT=8080
```

### 3. Start PostgreSQL Database

**Using Docker**:
```bash
docker run -d \
  --name alicia-postgres \
  -e POSTGRES_USER=alicia \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=alicia \
  -p 5432:5432 \
  pgvector/pgvector:pg15
```

**Or use existing PostgreSQL** and create database:
```bash
createdb alicia
psql -d alicia -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

### 4. Run Database Migrations

Apply schema migrations:

```bash
# From repository root
migrate -path migrations -database "$ALICIA_POSTGRES_URL" up
```

Verify migrations applied:
```bash
migrate -path migrations -database "$ALICIA_POSTGRES_URL" version
```

### 5. Build Backend Binary

```bash
# Install dependencies
go mod download

# Build binary
go build -o bin/alicia ./cmd/alicia
```

### 6. Start Backend Server

The server provides the REST API for conversation management:

```bash
./bin/alicia serve
```

Expected output:
```
2025-12-30T12:00:00Z INFO  Starting Alicia server
2025-12-30T12:00:00Z INFO  Database connected
2025-12-30T12:00:00Z INFO  LiveKit service initialized url=ws://localhost:7880
2025-12-30T12:00:00Z INFO  Server listening addr=0.0.0.0:8080
```

Verify server is running:
```bash
curl http://localhost:8080/api/health
# Should return: {"status":"ok"}
```

### 7. Start Agent Process

The agent handles real-time conversation processing (ASR, LLM, TTS):

**In a new terminal**:
```bash
./bin/alicia agent
```

Expected output:
```
2025-12-30T12:00:00Z INFO  Starting Alicia agent
2025-12-30T12:00:00Z INFO  Connected to LiveKit url=ws://localhost:7880
2025-12-30T12:00:00Z INFO  ASR service initialized url=http://localhost:8765/v1
2025-12-30T12:00:00Z INFO  TTS service initialized url=http://localhost:8765/v1
2025-12-30T12:00:00Z INFO  LLM service initialized url=http://localhost:8000/v1
2025-12-30T12:00:00Z INFO  Agent ready, waiting for rooms...
```

### 8. Start Frontend (Web UI)

**In a new terminal**:

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies (first time only)
npm install

# Start development server
npm run dev
```

Expected output:
```
VITE v5.0.0  ready in 500 ms

  ➜  Local:   http://localhost:3000/
  ➜  Network: use --host to expose
```

### 9. Open Web UI

Open your browser to [http://localhost:3000](http://localhost:3000)

You should see the Alicia web interface.

## Your First Conversation

### Create a New Conversation

1. Click **"New Conversation"** button in the UI
2. (Optional) Click settings icon to configure:
   - TTS voice
   - Enable/disable memory
   - Enable/disable reasoning
3. The conversation view opens

### Start Talking

**Option 1: Voice input**
1. Click the microphone button
2. Grant browser microphone permission (if prompted)
3. Speak your message
4. Wait for the agent to respond with synthesized voice

**Option 2: Text input**
1. Type your message in the text box
2. Press Enter or click Send
3. The agent will respond (text + optional voice)

### Example Interactions

**Simple query**:
```
You: What's the capital of France?
Alicia: The capital of France is Paris.
```

**Follow-up conversation** (tests memory):
```
You: What's interesting about Paris?
Alicia: Paris is known for landmarks like the Eiffel Tower...

You: When was it built?  # "it" refers to Eiffel Tower from context
Alicia: The Eiffel Tower was built between 1887 and 1889...
```

**Tool usage** (if tools enabled):
```
You: What's the weather like today?
Alicia: [Uses weather tool] The current temperature is 72°F...
```

### Interrupting the Agent

While the agent is speaking:
- **Web**: Click the "Stop" button
- **Voice**: Start speaking (agent detects interruption and stops)

### Managing Conversations

**Archive** (pause without deleting):
- Click the archive icon in conversation list
- Resume later by clicking the conversation again

**Delete** (permanent):
- Click the delete icon
- Confirm deletion
- Conversation is soft-deleted (recoverable from database if needed)

## Verifying the Setup

### Check All Services Running

Use this checklist to verify everything is working:

- [ ] PostgreSQL is accessible: `psql $ALICIA_POSTGRES_URL -c "SELECT 1;"`
- [ ] Backend server responds: `curl http://localhost:8080/api/health`
- [ ] LiveKit is running: Check [http://localhost:7880](http://localhost:7880) (if using local LiveKit)
- [ ] LLM server responds: `curl http://localhost:8000/v1/models`
- [ ] ASR/TTS server responds: `curl http://localhost:8765/health`
- [ ] Agent process shows "Agent ready" in logs
- [ ] Frontend loads at [http://localhost:3000](http://localhost:3000)

### Test Message Flow

**Create test conversation**:
```bash
curl -X POST http://localhost:8080/api/conversations \
  -H "Content-Type: application/json" \
  -d '{"title": "Test conversation"}'
```

Should return conversation object with `id` and `livekit_room_name`.

### Check Database

Verify data is persisted:

```bash
psql $ALICIA_POSTGRES_URL
```

```sql
-- List conversations
SELECT id, title, status FROM conversations;

-- List messages
SELECT id, conversation_id, role, contents FROM messages LIMIT 10;
```

## Common Issues

### Issue: Database Connection Failed

**Symptom**:
```
FATAL: database connection failed
```

**Solutions**:
1. Verify PostgreSQL is running: `pg_isready`
2. Check connection string in `.env` is correct
3. Ensure database `alicia` exists: `psql -l | grep alicia`
4. Verify pgvector extension installed: `psql -d alicia -c "\dx vector"`

### Issue: Missing Environment Variables

**Symptom**:
```
ERROR: ALICIA_POSTGRES_URL is required
```

**Solutions**:
1. Ensure `.env` file exists: `ls -la .env`
2. Verify all required variables are set: `cat .env | grep ALICIA_`
3. Source environment manually if needed: `export $(cat .env | xargs)`

### Issue: LiveKit Connection Error

**Symptom**:
```
ERROR: failed to connect to LiveKit: connection refused
```

**Solutions**:
1. Check LiveKit server is running
2. Verify `ALICIA_LIVEKIT_URL` matches LiveKit server address
3. Test connectivity: `curl -v ws://localhost:7880`
4. Check API key/secret match LiveKit configuration

### Issue: No Audio from Agent

**Symptom**: Agent responds with text but no audio plays

**Solutions**:
1. Check browser granted microphone/speaker permissions
2. Verify TTS server is running: `curl http://localhost:8765/health`
3. Check agent logs for TTS errors: `./bin/alicia agent | grep -i tts`
4. Verify `ALICIA_TTS_VOICE` is a valid voice for your TTS model
5. Test TTS directly:
   ```bash
   curl -X POST http://localhost:8765/v1/audio/speech \
     -H "Content-Type: application/json" \
     -d '{"model": "hexgrad/kokoro-v0_19", "voice": "af_sky", "input": "Hello world"}'
   ```

### Issue: Agent Not Hearing User

**Symptom**: User speaks but agent doesn't transcribe

**Solutions**:
1. Check microphone is selected in browser
2. Verify audio track is published (check browser console)
3. Check agent subscribed to audio track (agent logs)
4. Verify ASR server is running: `curl http://localhost:8765/health`
5. Check silence detection threshold (may be too high)

### Issue: LLM Errors

**Symptom**:
```
ERROR: failed to generate response: LLM service unavailable
```

**Solutions**:
1. Verify LLM server is running: `curl http://localhost:8000/v1/models`
2. Check model name matches: `ALICIA_LLM_MODEL=Qwen/Qwen2.5-7B-Instruct-AWQ`
3. Ensure LLM server has enough VRAM/memory
4. Check LLM server logs for errors
5. Verify API key if required: `ALICIA_LLM_API_KEY=...`

### Issue: Frontend Build Errors

**Symptom**:
```
ERROR: Cannot find module '@livekit/components-react'
```

**Solutions**:
1. Delete `node_modules` and reinstall:
   ```bash
   cd frontend
   rm -rf node_modules package-lock.json
   npm install
   ```
2. Ensure Node.js version is 20+: `node --version`
3. Clear npm cache: `npm cache clean --force`

### Issue: Port Already in Use

**Symptom**:
```
ERROR: bind: address already in use
```

**Solutions**:
1. Find process using port: `lsof -i :8080` (or whatever port)
2. Kill process: `kill -9 <PID>`
3. Or change port in `.env`: `ALICIA_SERVER_PORT=8081`

## Architecture Quick Reference

```
┌─────────────────────────────────────────────────────────┐
│                     User Browser                        │
│  ┌───────────────────────────────────────────────────┐  │
│  │     React Frontend (localhost:3000)               │  │
│  │  - LiveKit Client SDK                             │  │
│  │  - Audio track (microphone)                       │  │
│  │  - Data channel (protocol messages)               │  │
│  └────────────┬─────────────────────┬─────────────────┘  │
└───────────────┼─────────────────────┼────────────────────┘
                │                     │
         REST API (JSON)      LiveKit WebSocket
                │                     │
                ▼                     ▼
    ┌────────────────────┐  ┌──────────────────┐
    │  Backend Server    │  │  LiveKit Server  │
    │  (localhost:8080)  │  │  (localhost:7880)│
    │  - REST API        │  │  - WebRTC        │
    │  - DB operations   │  │  - Audio tracks  │
    └─────────┬──────────┘  └────────┬─────────┘
              │                      │
              │                      ▼
              │            ┌──────────────────┐
              │            │  Alicia Agent    │
              │            │  - ASR (Whisper) │
              │            │  - LLM (Qwen)    │
              │            │  - TTS (Kokoro)  │
              │            └────────┬─────────┘
              │                     │
              ▼                     ▼
    ┌──────────────────────────────────────┐
    │  PostgreSQL + pgvector               │
    │  - Conversations                     │
    │  - Messages                          │
    │  - Memories (vector embeddings)      │
    └──────────────────────────────────────┘
```

## Next Steps

Now that Alicia is running, explore these advanced features:

1. **Memory System**: Create memories for personalization
   - See [Optimization System](OPTIMIZATION_SYSTEM.md)

2. **Tool Integration**: Add custom MCP tools
   - See [Components](COMPONENTS.md) for tool architecture

3. **GEPA Optimization**: Tune response dimensions
   - See [GEPA Primer](GEPA_PRIMER.md)

4. **Mobile Client**: Try the Android app
   - Build instructions in `/android/README.md`

5. **Production Deployment**: Deploy to cloud
   - See [Deployment Guide](DEPLOYMENT.md)

## Development Workflow

### Making Changes

**Backend changes**:
```bash
# Edit Go code
# Rebuild
go build -o bin/alicia ./cmd/alicia

# Restart server
./bin/alicia serve
```

**Frontend changes**:
```bash
# Edit TypeScript/React code
# Vite hot-reloads automatically (no restart needed)
```

**Database schema changes**:
```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_table

# Edit migrations/XXXXXX_add_new_table.up.sql
# Apply migration
migrate -path migrations -database "$ALICIA_POSTGRES_URL" up
```

### Running Tests

**Backend tests**:
```bash
go test ./...
```

**Frontend tests**:
```bash
cd frontend
npm test
```

### Linting

**Go**:
```bash
golangci-lint run
```

**TypeScript**:
```bash
cd frontend
npm run lint
```

## Troubleshooting Resources

- **Logs**: Check terminal outputs for errors
- **LiveKit Inspector**: [http://localhost:7880/inspector](http://localhost:7880/inspector) (if running local LiveKit)
- **Browser DevTools**: Check console for frontend errors
- **Database**: Query directly with `psql` to inspect data

## Getting Help

If you encounter issues not covered here:

1. **Check documentation**:
   - [Architecture](ARCHITECTURE.md) - System design
   - [Components](COMPONENTS.md) - Backend components
   - [LiveKit Integration](LIVEKIT.md) - Real-time communication
   - [Conversation Workflow](CONVERSATION_WORKFLOW.md) - Message flow

2. **File an issue**: [https://github.com/longregen/alicia/issues](https://github.com/longregen/alicia/issues)

3. **Review logs**: Look for ERROR or FATAL messages in terminal output

## See Also

- [Architecture Overview](ARCHITECTURE.md) - Detailed system architecture
- [LiveKit Integration](LIVEKIT.md) - Real-time communication deep-dive
- [Conversation Workflow](CONVERSATION_WORKFLOW.md) - Message flow and state transitions
- [Database Schema](DATABASE.md) - Data models and queries
- [Deployment Guide](DEPLOYMENT.md) - Production deployment
- [User Stories](USER_STORIES.md) - Feature documentation
