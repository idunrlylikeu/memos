# Production Build Guide

## How to build and run the production Docker image

### Single Command Build & Run

```bash
docker-compose -f docker-compose.prod.yml up -d
```

That's it! Compose will automatically:
1. Build the frontend
2. Build the backend binary
3. Start the container

### Building only (without running)

```bash
docker-compose -f docker-compose.prod.yml build
```

### Manual Docker Build

If you prefer to build without compose:

```bash
# Build the image
docker build -f docker/backend/Dockerfile.prod -t memos:latest .

# Run the container
docker run -d \
  --name memos-prod \
  -p 5230:5230 \
  -v memos-data:/var/opt/memos \
  -e TZ=UTC \
  memos:latest
```

## What the build does

The production Dockerfile has a fully self-contained 3-stage build:

1. **Stage 1 - Frontend Build** (`frontend-builder`)
   - Installs Node.js dependencies (pnpm)
   - Builds the React/TypeScript frontend
   - Outputs to `/app/dist`

2. **Stage 2 - Backend Build** (`builder`)
   - Copies frontend dist files
   - Downloads Go dependencies
   - Builds the Go binary with embedded frontend
   - Outputs `memos` binary

3. **Stage 3 - Runtime** (final stage)
   - Minimal Alpine Linux image
   - Copies binary and entrypoint script
   - Runs as non-root user (uid 10001)
   - Exposes port 5230

## Common Commands

```bash
# Build and start
docker-compose -f docker-compose.prod.yml up -d

# Stop
docker-compose -f docker-compose.prod.yml down

# View logs
docker-compose -f docker-compose.prod.yml logs -f

# Restart
docker-compose -f docker-compose.prod.yml restart

# Rebuild from scratch (clear cache)
docker-compose -f docker-compose.prod.yml build --no-cache
```

## Environment Variables

For additional configuration, create a `.env` file in the repo root:

```env
# Optional: AI features  
OPENROUTER_API_KEY=sk-or-...
AI_MODEL=openai/gpt-4o

# Optional: Timezone (default: UTC)
TZ=UTC

# Optional: Port (default: 5230)
MEMOS_PORT=5230
```

## Data Persistence

The container mounts `memos-data` volume at `/var/opt/memos`:
- SQLite database: `/var/opt/memos/memos.db`
- Uploaded files: `/var/opt/memos/`

To backup:

```bash
docker run --rm -v memos-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/memos-backup.tar.gz /data
```

To restore:

```bash
docker run --rm -v memos-data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/memos-backup.tar.gz -C /
```

## Troubleshooting

**Build fails with missing dependencies:**
```bash
# Clear build cache and retry
docker builder prune --all
docker-compose -f docker-compose.prod.yml build --no-cache
```

**Port already in use:**
Edit `docker-compose.prod.yml` and change:
```yaml
ports:
  - "5231:5230"  # Use 5231 instead of 5230
```

**Container exits immediately:**
```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs memos
```

**Need to rebuild after code changes:**
```bash
# Full rebuild (slow, clears all cache)
docker-compose -f docker-compose.prod.yml up -d --build

# Or rebuild without cache
docker-compose -f docker-compose.prod.yml build --no-cache && \
docker-compose -f docker-compose.prod.yml up -d
```
