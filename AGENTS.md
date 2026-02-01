# Memos Codebase Guide for AI Agents

This document provides comprehensive guidance for AI agents working with the Memos codebase. It covers architecture, workflows, conventions, and key patterns.

## Project Overview

Memos is a self-hosted knowledge management platform built with:
- **Backend:** Go 1.25 with gRPC + Connect RPC
- **Frontend:** React 18.3 + TypeScript + Vite 7
- **Databases:** SQLite (default), MySQL, PostgreSQL
- **Protocol:** Protocol Buffers (v2) with buf for code generation
- **API Layer:** Dual protocol - Connect RPC (browsers) + gRPC-Gateway (REST)

## Architecture

### Backend Architecture

```
cmd/memos/              # Entry point
└── main.go             # Cobra CLI, profile setup, server initialization

server/
├── server.go           # Echo HTTP server, healthz, background runners
├── auth/               # Authentication (JWT, PAT, session)
├── router/
│   ├── api/v1/        # gRPC service implementations
│   │   ├── v1.go      # Service registration, gateway & Connect setup
│   │   ├── acl_config.go   # Public endpoints whitelist
│   │   ├── connect_services.go  # Connect RPC handlers
│   │   ├── connect_interceptors.go # Auth, logging, recovery
│   │   └── *_service.go    # Individual services (memo, user, etc.)
│   ├── frontend/       # Static file serving (SPA)
│   ├── fileserver/     # Native HTTP file serving for media
│   └── rss/           # RSS feed generation
└── runner/
    ├── memopayload/    # Memo payload processing (tags, links, tasks)
    └── s3presign/     # S3 presigned URL management

store/                  # Data layer with caching
├── driver.go           # Driver interface (database operations)
├── store.go           # Store wrapper with cache layer
├── cache.go           # In-memory caching (instance settings, users)
├── migrator.go        # Database migrations
├── db/
│   ├── db.go          # Driver factory
│   ├── sqlite/        # SQLite implementation
│   ├── mysql/         # MySQL implementation
│   └── postgres/      # PostgreSQL implementation
└── migration/         # SQL migration files (embedded)

proto/                  # Protocol Buffer definitions
├── api/v1/           # API v1 service definitions
└── gen/               # Generated Go & TypeScript code
```

### Frontend Architecture

```
web/
├── src/
│   ├── components/     # React components
│   ├── contexts/       # React Context (client state)
│   │   ├── AuthContext.tsx      # Current user, auth state
│   │   ├── ViewContext.tsx      # Layout, sort order
│   │   └── MemoFilterContext.tsx # Filters, shortcuts
│   ├── hooks/          # React Query hooks (server state)
│   │   ├── useMemoQueries.ts    # Memo CRUD, pagination
│   │   ├── useUserQueries.ts    # User operations
│   │   ├── useAttachmentQueries.ts # Attachment operations
│   │   └── ...
│   ├── lib/            # Utilities
│   │   ├── query-client.ts  # React Query v5 client
│   │   └── connect.ts       # Connect RPC client setup
│   ├── pages/          # Page components
│   └── types/proto/    # Generated TypeScript from .proto
├── package.json        # Dependencies
└── vite.config.mts     # Vite config with dev proxy

### Docker Architecture

**Development Setup (docker-compose.dev.yml):**

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Network                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────────┐         ┌──────────────────┐     │
│  │  Frontend Dev    │         │  Backend Dev     │     │
│  │  Container       │────────▶│  Container       │     │
│  │                  │ Proxy  │                  │     │
│  │  - Vite dev      │  API   │  - Go with air   │     │
│  │  - Port 3001     │        │  - Hot reload    │     │
│  │  - pnpm dev      │        │  - Port 8081     │     │
│  │                  │        │  - SQLite        │     │
│  └──────────────────┘         └──────────────────┘     │
│         ▲                                                    │
│         │                                                    │
│         │ Hot reload (volume mounts)                        │
│         │                                                    │
│  ┌──────┴──────────────────────────────────────────────┐   │
│  │           Host Filesystem                           │   │
│  │  ./web/        → Frontend container /app            │   │
│  │  ./ (Go files) → Backend container /app              │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Development Containers:**

1. **Backend Container** (`docker/backend/Dockerfile.dev`):
   - Base: `golang:1.25-alpine`
   - Tools: `air` for hot reload
   - Volume mounts: Source code for live reload
   - Excludes: `web/node_modules`, `tmp/`
   - Port: 8081
   - Config: `.air.toml`

2. **Frontend Container** (`docker/frontend/Dockerfile.dev`):
   - Base: `node:20-alpine`
   - Package manager: `pnpm`
   - Volume mounts: `./web` directory
   - Environment: `DEV_PROXY_SERVER=http://backend:8081`
   - Port: 3001
   - Command: `pnpm dev --host`

**Production Setup (docker-compose.prod.yml):**

```
┌─────────────────────────────────┐
│   Single Production Container   │
├─────────────────────────────────┤
│                                 │
│  ┌───────────────────────────┐ │
│  │   Alpine Linux            │ │
│  │                           │ │
│  │  ┌─────────────────────┐  │ │
│  │  │ Go Binary           │  │ │
│  │  │ (with embedded      │  │ │
│  │  │  frontend files)    │  │ │
│  │  └─────────────────────┘  │ │
│  │                           │ │
│  │  SQLite DB /var/opt/memos │ │
│  └───────────────────────────┘ │
│                                 │
│  Port: 5230                     │
│  Volume: memos-data             │
└─────────────────────────────────┘
```

**Production Container:**
- Base: `alpine:3.21`
- Single Go binary with embedded frontend
- Data persistence via Docker volume
- Port: 5230 (default)
- User: `nonroot` (uid 10001)

**Docker File Structure:**

```
docker/
├── backend/
│   ├── Dockerfile.dev    # Development with air hot reload
│   │   - Installs air
│   │   - Mounts source code
│   │   - Runs air for live reload
│   │
│   └── Dockerfile.prod   # Production binary
│       - Multi-stage build
│       - Embeds frontend dist
│       - Minimal alpine runtime
│
└── frontend/
    ├── Dockerfile.dev    # Development server
    │   - Installs pnpm
    │   - Mounts web/ directory
    │   - Runs vite dev server
    │
    └── Dockerfile.prod   # Production build
        - Runs pnpm build
        - Outputs to ./dist
        - Used by backend Dockerfile.prod
```

**Hot Reload Configuration (`.air.toml`):**

```toml
[build]
  cmd = "go build -o ./tmp/memos ./cmd/memos"
  bin = "tmp/memos"
  include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml"]
  exclude_dir = ["web", "tmp", "vendor", "node_modules", "dist"]
  delay = 1000  # ms before rebuild
```

**Volume Mount Strategy:**

Development volumes use "anonymous volumes" to avoid conflicts:
- `/app/web/node_modules` - Container's own node_modules
- `/app/tmp` - Air build artifacts
- `go-mod-cache` - Go module cache for faster rebuilds

This prevents host node_modules from conflicting with container architecture.

plugin/                 # Backend plugins
├── scheduler/         # Cron jobs
├── email/            # Email delivery
├── filter/           # CEL filter expressions
├── webhook/          # Webhook dispatch
├── markdown/         # Markdown parsing & rendering
├── httpgetter/        # HTTP fetching (metadata, images)
└── storage/s3/       # S3 storage backend

docker/                 # Docker configurations
├── backend/
│   ├── Dockerfile.dev  # Development with air hot reload
│   └── Dockerfile.prod # Production build
└── frontend/
    ├── Dockerfile.dev  # Vite dev server
    └── Dockerfile.prod # Frontend build stage
```

## Key Architectural Patterns

### 1. API Layer: Dual Protocol

**Connect RPC (Browser Clients):**
- Protocol: `connectrpc.com/connect`
- Base path: `/memos.api.v1.*`
- Interceptor chain: Metadata → Logging → Recovery → Auth
- Returns type-safe responses to React frontend
- See: `server/router/api/v1/connect_interceptors.go:177-227`

**gRPC-Gateway (REST API):**
- Protocol: Standard HTTP/JSON
- Base path: `/api/v1/*`
- Uses same service implementations as Connect
- Useful for external tools, CLI clients
- See: `server/router/api/v1/v1.go:52-96`

**Authentication:**
- JWT Access Tokens (V2): Stateless, 15-min expiration, verified via `AuthenticateByAccessTokenV2`
- Personal Access Tokens (PAT): Stateful, long-lived, validated against database
- Both use `Authorization: Bearer <token>` header
- See: `server/auth/authenticator.go:17-166`

### 2. Store Layer: Interface Pattern

All database operations go through the `Driver` interface:
```go
type Driver interface {
    GetDB() *sql.DB
    Close() error

    IsInitialized(ctx context.Context) (bool, error)

    CreateMemo(ctx context.Context, create *Memo) (*Memo, error)
    ListMemos(ctx context.Context, find *FindMemo) ([]*Memo, error)
    UpdateMemo(ctx context.Context, update *UpdateMemo) error
    DeleteMemo(ctx context.Context, delete *DeleteMemo) error

    // ... similar methods for all resources
}
```

**Three Implementations:**
- `store/db/sqlite/` - SQLite (modernc.org/sqlite)
- `store/db/mysql/` - MySQL (go-sql-driver/mysql)
- `store/db/postgres/` - PostgreSQL (lib/pq)

**Caching Strategy:**
- Store wrapper maintains in-memory caches for:
  - Instance settings (`instanceSettingCache`)
  - Users (`userCache`)
  - User settings (`userSettingCache`)
- Config: Default TTL 10 min, cleanup interval 5 min, max 1000 items
- See: `store/store.go:10-57`

### 3. Frontend State Management

**React Query v5 (Server State):**
- All API calls go through custom hooks in `web/src/hooks/`
- Query keys organized by resource: `memoKeys`, `userKeys`, `attachmentKeys`
- Default staleTime: 30s, gcTime: 5min
- Automatic refetch on window focus, reconnect
- See: `web/src/lib/query-client.ts`

**React Context (Client State):**
- `AuthContext`: Current user, auth initialization, logout
- `ViewContext`: Layout mode (LIST/MASONRY), sort order
- `MemoFilterContext`: Active filters, shortcut selection, URL sync

### 4. Database Migration System

**Migration Flow:**
1. `preMigrate`: Check if DB exists. If not, apply `LATEST.sql`
2. `checkMinimumUpgradeVersion`: Reject pre-0.22 installations
3. `applyMigrations`: Apply incremental migrations in single transaction
4. Demo mode: Seed with demo data

**Schema Versioning:**
- Stored in `system_setting` table
- Format: `major.minor.patch`
- Migration files: `store/migration/{driver}/{version}/NN__description.sql`
- See: `store/migrator.go:21-414`

### 5. Protocol Buffer Code Generation

**Definition Location:** `proto/api/v1/*.proto`

**Regeneration:**
```bash
cd proto && buf generate
```

**Generated Outputs:**
- Go: `proto/gen/api/v1/` (used by backend services)
- TypeScript: `web/src/types/proto/api/v1/` (used by frontend)

**Linting:** `proto/buf.yaml` - BASIC lint rules, FILE breaking changes

## Development Commands

### Backend

```bash
# Start dev server
go run ./cmd/memos --port 8081

# Run all tests
go test ./...

# Run tests for specific package
go test ./store/...
go test ./server/router/api/v1/test/...

# Lint (golangci-lint)
golangci-lint run

# Format imports
goimports -w .

# Run with MySQL/Postgres
DRIVER=mysql go run ./cmd/memos
DRIVER=postgres go run ./cmd/memos
```

### Frontend

```bash
# Install dependencies
cd web && pnpm install

# Start dev server (proxies API to localhost:8081)
pnpm dev

# Type checking
pnpm lint

# Auto-fix lint issues
pnpm lint:fix

# Format code
pnpm format

# Build for production
pnpm build

# Build and copy to backend
pnpm release
```

### Docker Development

```bash
# Development with hot reload (frontend + backend)
docker-compose -f docker-compose.dev.yml up

# Development - backend only
docker-compose -f docker-compose.dev.yml up backend

# Development - frontend only
docker-compose -f docker-compose.dev.yml up frontend

# Production build and run
docker-compose -f docker-compose.prod.yml up -d

# View logs
docker-compose -f docker-compose.dev.yml logs -f

# Stop containers
docker-compose -f docker-compose.dev.yml down

# Rebuild after changes
docker-compose -f docker-compose.dev.yml build --no-cache
```

**Development Ports:**
- Frontend (Vite hot reload): http://localhost:3001
- Backend API: http://localhost:8081

**Production Port:**
- Single container (embedded frontend): http://localhost:5230

### Protocol Buffers

```bash
# Regenerate Go and TypeScript from .proto files
cd proto && buf generate

# Lint proto files
cd proto && buf lint

# Check for breaking changes
cd proto && buf breaking --against .git#main
```

## Key Workflows

### Adding a New API Endpoint

1. **Define in Protocol Buffer:**
   - Edit `proto/api/v1/*_service.proto`
   - Add request/response messages
   - Add RPC method to service

2. **Regenerate Code:**
   ```bash
   cd proto && buf generate
   ```

3. **Implement Service (Backend):**
   - Add method to `server/router/api/v1/*_service.go`
   - Follow existing patterns: fetch user, validate, call store
   - Add Connect wrapper to `server/router/api/v1/connect_services.go` (optional, same implementation)

4. **If Public Endpoint:**
   - Add to `server/router/api/v1/acl_config.go:11-34`

5. **Create Frontend Hook (if needed):**
   - Add query/mutation to `web/src/hooks/use*Queries.ts`
   - Use existing query key factories

### Database Schema Changes

1. **Create Migration Files:**
   ```
   store/migration/sqlite/0.28/1__add_new_column.sql
   store/migration/mysql/0.28/1__add_new_column.sql
   store/migration/postgres/0.28/1__add_new_column.sql
   ```

2. **Update LATEST.sql:**
   - Add change to `store/migration/{driver}/LATEST.sql`

3. **Update Store Interface (if new table/model):**
   - Add methods to `store/driver.go:8-71`
   - Implement in `store/db/{driver}/*.go`

4. **Test Migration:**
   - Run `go test ./store/test/...` to verify

### Docker Development Workflow

**1. Initial Setup:**
```bash
# Clone repository
git clone https://github.com/usememos/memos.git
cd memos

# Start development environment
docker-compose -f docker-compose.dev.yml up
```

**2. Making Changes:**

- **Backend changes (Go files):**
  - Edit any `*.go` file in the repository
  - Air detects changes → auto-rebuilds → restarts server
  - Changes reflected in ~2-3 seconds
  - Logs show: `[air] Building...` → `[air] Running...`

- **Frontend changes (React/TS files):**
  - Edit files in `web/src/` or `web/*.tsx`
  - Vite HMR pushes updates to browser
  - Changes reflected instantly (no full refresh for CSS/styled components)
  - Component changes trigger fast refresh

**3. Debugging:**

```bash
# Backend logs
docker-compose -f docker-compose.dev.yml logs -f backend

# Frontend logs
docker-compose -f docker-compose.dev.yml logs -f frontend

# Enter container for debugging
docker-compose -f docker-compose.dev.yml exec backend sh
docker-compose -f docker-compose.dev.yml exec frontend sh
```

**4. Testing Changes:**

- Backend API: http://localhost:8081/api/v1/status
- Frontend UI: http://localhost:3001
- Both containers share the same SQLite database

**5. Building for Production:**

```bash
# Stop dev environment
docker-compose -f docker-compose.dev.yml down

# Build and start production
docker-compose -f docker-compose.prod.yml up -d

# Access at http://localhost:5230
```

**6. Production Data Backup:**

```bash
# Create backup
docker run --rm \
  -v memos_memos-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/memos-backup-$(date +%Y%m%d).tar.gz /data

# List backups
ls -lh memos-backup-*.tar.gz
```

### Adding a New Frontend Page

1. **Create Page Component:**
   - Add to `web/src/pages/NewPage.tsx`
   - Use existing hooks for data fetching

2. **Add Route:**
   - Edit `web/src/App.tsx` (or router configuration)

3. **Use React Query:**
   ```typescript
   import { useMemos } from "@/hooks/useMemoQueries";
   const { data, isLoading } = useMemos({ filter: "..." });
   ```

4. **Use Context for Client State:**
   ```typescript
   import { useView } from "@/contexts/ViewContext";
   const { layout, toggleSortOrder } = useView();
   ```

## Testing

### Backend Tests

**Test Pattern:**
```go
func TestMemoCreation(t *testing.T) {
    ctx := context.Background()
    store := test.NewTestingStore(ctx, t)

    // Create test user
    user, _ := createTestUser(ctx, store, t)

    // Execute operation
    memo, err := store.CreateMemo(ctx, &store.Memo{
        CreatorID: user.ID,
        Content:  "Test memo",
        // ...
    })
    require.NoError(t, err)
    assert.NotNil(t, memo)
}
```

**Test Utilities:**
- `store/test/store.go:22-35` - `NewTestingStore()` creates isolated DB
- `store/test/store.go:37-77` - `resetTestingDB()` cleans tables
- Test DB determined by `DRIVER` env var (default: sqlite)

**Running Tests:**
```bash
# All tests
go test ./...

# Specific package
go test ./store/...
go test ./server/router/api/v1/test/...

# With coverage
go test -cover ./...
```

### Frontend Testing

**TypeScript Checking:**
```bash
cd web && pnpm lint
```

**No Automated Tests:**
- Frontend relies on TypeScript checking and manual validation
- React Query DevTools available in dev mode (bottom-left)

## Code Conventions

### Go

**Error Handling:**
- Use `github.com/pkg/errors` for wrapping: `errors.Wrap(err, "context")`
- Return structured gRPC errors: `status.Errorf(codes.NotFound, "message")`

**Naming:**
- Package names: lowercase, single word (e.g., `store`, `server`)
- Interfaces: `Driver`, `Store`, `Service`
- Methods: PascalCase for exported, camelCase for internal

**Comments:**
- Public exported functions must have comments (godot enforces)
- Use `//` for single-line, `/* */` for multi-line

**Imports:**
- Grouped: stdlib, third-party, local
- Sorted alphabetically within groups
- Use `goimports -w .` to format

### TypeScript/React

**Components:**
- Functional components with hooks
- Use `useMemo`, `useCallback` for optimization
- Props interfaces: `interface Props { ... }`

**State Management:**
- Server state: React Query hooks
- Client state: React Context
- Avoid direct useState for server data

**Styling:**
- Tailwind CSS v4 via `@tailwindcss/vite`
- Use `clsx` and `tailwind-merge` for conditional classes

**Imports:**
- Absolute imports with `@/` alias
- Group: React, third-party, local
- Auto-organized by Biome

## Important Files Reference

### Backend Entry Points

| File | Purpose |
|------|---------|
| `cmd/memos/main.go` | Server entry point, CLI setup |
| `server/server.go` | Echo server initialization, background runners |
| `store/store.go` | Store wrapper with caching |
| `store/driver.go` | Database driver interface |
| `scripts/Dockerfile` | Original production Dockerfile (legacy) |
| `docker/backend/Dockerfile.dev` | Development backend with air hot reload |
| `docker/backend/Dockerfile.prod` | Production backend (recommended) |
| `docker/frontend/Dockerfile.dev` | Development frontend with Vite |
| `docker/frontend/Dockerfile.prod` | Frontend build stage |
| `docker-compose.dev.yml` | Development orchestration |
| `docker-compose.prod.yml` | Production orchestration |
| `.air.toml` | Air hot reload configuration |

### API Layer

| File | Purpose |
|------|---------|
| `server/router/api/v1/v1.go` | Service registration, gateway setup |
| `server/router/api/v1/acl_config.go` | Public endpoints whitelist |
| `server/router/api/v1/connect_interceptors.go` | Connect interceptors |
| `server/auth/authenticator.go` | Authentication logic |

### Frontend Core

| File | Purpose |
|------|---------|
| `web/src/lib/query-client.ts` | React Query client configuration |
| `web/src/contexts/AuthContext.tsx` | User authentication state |
| `web/src/contexts/ViewContext.tsx` | UI preferences |
| `web/src/contexts/MemoFilterContext.tsx` | Filter state |
| `web/src/hooks/useMemoQueries.ts` | Memo queries/mutations |

### Data Layer

| File | Purpose |
|------|---------|
| `store/memo.go` | Memo model definitions, store methods |
| `store/user.go` | User model definitions |
| `store/attachment.go` | Attachment model definitions |
| `store/migrator.go` | Migration logic |
| `store/db/db.go` | Driver factory |
| `store/db/sqlite/sqlite.go` | SQLite driver implementation |

## Configuration

### Backend Environment Variables

| Variable | Default | Description |
|----------|----------|-------------|
| `MEMOS_DEMO` | `false` | Enable demo mode |
| `MEMOS_PORT` | `8081` | HTTP port |
| `MEMOS_ADDR` | `` | Bind address (empty = all) |
| `MEMOS_DATA` | `~/.memos` | Data directory |
| `MEMOS_DRIVER` | `sqlite` | Database: `sqlite`, `mysql`, `postgres` |
| `MEMOS_DSN` | `` | Database connection string |
| `MEMOS_INSTANCE_URL` | `` | Instance base URL |

### Frontend Environment Variables

| Variable | Default | Description |
|----------|----------|-------------|
| `DEV_PROXY_SERVER` | `http://localhost:8081` | Backend proxy target |

### Docker Environment Variables

| Variable | Default | Description |
|----------|----------|-------------|
| `MEMOS_PORT` | `8081` (dev) / `5230` (prod) | HTTP port |
| `TZ` | `UTC` | Timezone |
| `DEV_PROXY_SERVER` | `http://backend:8081` | Backend URL (dev frontend) |

### Docker Volumes

| Volume | Purpose | Persistence |
|--------|---------|-------------|
| `go-mod-cache` | Go module dependencies | Across container rebuilds |
| `memos-data` | SQLite database + uploads | Production data persistence |
| `/app/node_modules` | Frontend dependencies | Container-only (prevents host conflicts) |
| `/app/tmp` | Air build artifacts | Container-only (hot reload) |

### Database Behavior in Docker

**Development (SQLite):**
- Location: `/app/var/memos.db` (inside backend container)
- All users share the same database file
- Suitable for: Local development, single-user testing
- Not persisted outside container (unless volume mounted)

**Production (SQLite):**
- Location: `/var/opt/memos/memos.db` (inside container)
- Persisted via Docker volume `memos-data`
- All users accessing your instance share the same database
- Suitable for: Personal instances, small teams, low-to-medium traffic
- For high concurrent users, consider PostgreSQL instead

**Database Access:**
- SQLite is a **single shared database** for all users
- If you deploy Memos to a server, every user who accesses your web app connects to the same SQLite file
- This is different from per-user databases - it's a multi-tenant system with shared data
- Access control is handled at the application layer (user authentication, permissions, ACLs)

## CI/CD

### GitHub Workflows

**Backend Tests** (`.github/workflows/backend-tests.yml`):
- Runs on `go.mod`, `go.sum`, `**.go` changes
- Steps: verify `go mod tidy`, golangci-lint, all tests

**Frontend Tests** (`.github/workflows/frontend-tests.yml`):
- Runs on `web/**` changes
- Steps: pnpm install, lint, build

**Proto Lint** (`.github/workflows/proto-linter.yml`):
- Runs on `.proto` changes
- Steps: buf lint, buf breaking check

### Linting Configuration

**Go** (`.golangci.yaml`):
- Linters: revive, govet, staticcheck, misspell, gocritic, etc.
- Formatter: goimports
- Forbidden: `fmt.Errorf`, `ioutil.ReadDir`

**TypeScript** (`web/biome.json`):
- Linting: Biome (ESLint replacement)
- Formatting: Biome (Prettier replacement)
- Line width: 140 characters
- Semicolons: always

## Common Tasks

### Docker Quick Reference

```bash
# Development - Start both services
docker-compose -f docker-compose.dev.yml up

# Development - Start in detached mode
docker-compose -f docker-compose.dev.yml up -d

# Development - View logs
docker-compose -f docker-compose.dev.yml logs -f backend
docker-compose -f docker-compose.dev.yml logs -f frontend

# Development - Restart with changes
docker-compose -f docker-compose.dev.yml restart backend

# Development - Stop all
docker-compose -f docker-compose.dev.yml down

# Development - Rebuild (after dependency changes)
docker-compose -f docker-compose.dev.yml build --no-cache backend
docker-compose -f docker-compose.dev.yml build --no-cache frontend

# Production - Build and start
docker-compose -f docker-compose.prod.yml up -d

# Production - View logs
docker-compose -f docker-compose.prod.yml logs -f

# Production - Stop
docker-compose -f docker-compose.prod.yml down

# Production - Backup data
docker run --rm -v memos_memos-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/memos-backup.tar.gz /data

# Production - Restore data
docker run --rm -v memos_memos-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/memos-backup.tar.gz -C /
```

### Debugging API Issues

1. Check Connect interceptor logs: `server/router/api/v1/connect_interceptors.go:79-105`
2. Verify endpoint is in `acl_config.go` if public
3. Check authentication via `auth/authenticator.go:133-165`
4. Test with curl: `curl -H "Authorization: Bearer <token>" http://localhost:8081/api/v1/...`

### Debugging Frontend State

1. Open React Query DevTools (bottom-left in dev)
2. Inspect query cache, mutations, refetch behavior
3. Check Context state via React DevTools
4. Verify filter state in MemoFilterContext

### Running Tests Against Multiple Databases

```bash
# SQLite (default)
DRIVER=sqlite go test ./...

# MySQL (requires running MySQL server)
DRIVER=mysql DSN="user:pass@tcp(localhost:3306)/memos" go test ./...

# PostgreSQL (requires running PostgreSQL server)
DRIVER=postgres DSN="postgres://user:pass@localhost:5432/memos" go test ./...
```

### Docker Troubleshooting

**Hot Reload Not Working:**

1. **Backend (air not detecting changes):**
   - Check volume mounts: `docker-compose -f docker-compose.dev.yml config`
   - Verify `.air.toml` is in the root directory
   - Check air logs: `docker-compose -f docker-compose.dev.yml logs backend`
   - Ensure exclude_dir doesn't include your changed files

2. **Frontend (Vite not detecting changes):**
   - Verify `./web` is mounted to `/app`
   - Check that `/app/node_modules` is an anonymous volume (prevents host conflicts)
   - Look for Vite's HMR logs in container output

**Container Build Issues:**

1. **Slow builds:**
   - Go module cache should speed up after first build
   - Use `--no-cache` only if dependencies changed
   - Pre-build frontend locally: `cd web && pnpm install`

2. **Architecture mismatches:**
   - Apple Silicon users: No action needed (auto-detected)
   - BuildKit multi-platform support enabled by default

**Port Conflicts:**

```bash
# Change ports in docker-compose.dev.yml:
services:
  backend:
    ports:
      - "9081:8081"  # Use 9081 instead of 8081
  frontend:
    ports:
      - "4001:3001"  # Use 4001 instead of 3001
```

**Database Persistence:**

```bash
# Check SQLite database location
docker-compose -f docker-compose.prod.yml exec memos ls -la /var/opt/memos/

# Backup SQLite database
docker-compose -f docker-compose.prod.yml exec memos cp /var/opt/memos/memos.db /var/opt/memos/backup.db

# Restore from volume
docker volume inspect memos_memos-data
```

## Plugin System

Backend supports pluggable components in `plugin/`:

| Plugin | Purpose |
|--------|----------|
| `scheduler` | Cron-based job scheduling |
| `email` | SMTP email delivery |
| `filter` | CEL expression filtering |
| `webhook` | HTTP webhook dispatch |
| `markdown` | Markdown parsing (goldmark) |
| `httpgetter` | HTTP content fetching |
| `storage/s3` | S3-compatible storage |

Each plugin has its own README with usage examples.

## Performance Considerations

### Backend

- Database queries use pagination (`limit`, `offset`)
- In-memory caching reduces DB hits for frequently accessed data
- WAL journal mode for SQLite (reduces locking)
- Thumbnail generation limited to 3 concurrent operations

### Frontend

- React Query reduces redundant API calls
- Infinite queries for large lists (pagination)
- Manual chunks: `utils-vendor`, `mermaid-vendor`, `leaflet-vendor`
- Lazy loading for heavy components

## Security Notes

- JWT secrets must be kept secret (generated on first run in production mode)
- Personal Access Tokens stored as SHA-256 hashes in database
- CSRF protection via SameSite cookies
- CORS enabled for all origins (configure for production)
- Input validation at service layer
- SQL injection prevention via parameterized queries
