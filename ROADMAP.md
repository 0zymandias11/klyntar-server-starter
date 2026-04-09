# klyntar — Build Roadmap

A CLI Reddit clone served over SSH, built in Go. Users `ssh` into the server and get a full terminal UI powered by Bubble Tea.

## Project Overview

Two sibling repositories:
- `klyntar/` — SSH server + Bubble Tea TUI (users `ssh` in, TUI rendered on server)
- `klyntar-server/` — gRPC backend (business logic, Postgres, Redis)

```
F:/doraemon/golang/
├── klyntar/         ← this repo: SSH server + TUI frontend
└── klyntar-server/  ← backend API server
```

---

## Phase 0 — Prerequisites & Tooling

**Install these before writing any code:**

| Tool | Install | Purpose |
|------|---------|---------|
| Go 1.23+ | https://go.dev/dl | Language |
| Docker Desktop | https://docker.com | Run Postgres + Redis locally |
| `buf` | `winget install bufbuild.buf` | Generate Go code from `.proto` files |
| `air` | `go install github.com/air-verse/air@latest` | Hot reload during dev |
| `golangci-lint` | https://golangci-lint.run/usage/install | Linter |
| `golang-migrate` CLI | https://github.com/golang-migrate/migrate | Run DB migrations from terminal |
| `evans` or `grpcurl` | `go install github.com/ktr0731/evans@latest` | Test gRPC endpoints |

**Go concepts to understand before Phase 1:**
- Packages and modules (`go mod init`, `import`)
- Structs, interfaces, methods on structs
- Error handling pattern (`if err != nil { return err }`)
- Goroutines and channels (needed for concurrent servers)
- `context.Context` — passed into almost every function in a Go server

---

## Phase 1 — Repo Skeletons

**Goal:** Two compilable Go modules with the correct folder layout. Nothing runs yet — just structure and `fmt.Println`.

### `klyntar-server/`
```
klyntar-server/
├── cmd/server/main.go      ← entry point
├── config/config.go        ← reads env vars (DATABASE_URL, GRPC_PORT, etc.)
├── go.mod
└── .env.example
```

`main.go` at this stage:
```go
package main

import "fmt"

func main() {
    fmt.Println("klyntar-server starting on :50051")
}
```

### `klyntar/`
```
klyntar/
├── main.go                 ← entry point
├── config/config.go        ← reads SSH_PORT, SERVER_ADDR, HOST_KEY_PATH
└── go.mod
```

**Go concepts for this phase:** `os.Getenv`, `fmt`, `log/slog`, `strconv.Atoi`

**Milestone:** `go run ./cmd/server` (server) and `go run .` (client) both compile and print their startup message.

---

## Phase 2 — Infrastructure (Docker + Database)

**Goal:** Postgres and Redis running locally via Docker. Full database schema defined in SQL migration files.

### Files to create
```
klyntar-server/
├── docker-compose.yml
├── Dockerfile
├── migrations/
│   ├── 001_users.up.sql       ← and .down.sql for each
│   ├── 002_voids.up.sql
│   ├── 003_posts.up.sql
│   ├── 004_comments.up.sql
│   ├── 005_votes.up.sql
│   └── 006_messages.up.sql
├── pkg/db/postgres.go         ← connect to Postgres (pgxpool)
└── pkg/cache/redis.go         ← connect to Redis
```

### Database schema (design these SQL files)

| Table | Key columns |
|-------|-------------|
| `users` | id (UUID), email (UNIQUE), username (UNIQUE), key_fingerprint (UNIQUE), device_mac, created_at |
| `user_keys` | id, user_id (FK→users), key_fingerprint (UNIQUE), device_name — *multi-device support* |
| `voids` | id, name (UNIQUE), description, creator_id (FK→users), created_at |
| `void_members` | user_id (FK), void_id (FK), joined_at — *composite PK* |
| `posts` | id, title, content, user_id (FK), void_id (FK), score (INT DEFAULT 0), created_at |
| `comments` | id, content, user_id (FK), post_id (FK), parent_id (FK→comments, nullable for nesting), created_at |
| `votes` | user_id (FK), post_id (FK nullable), comment_id (FK nullable), value (+1/-1), UNIQUE(user_id, post_id, comment_id) |
| `message_rooms` | id, name (nullable — null means DM), created_at |
| `room_members` | room_id (FK), user_id (FK) |
| `messages` | id, room_id (FK), user_id (FK), content, created_at |

**Go packages to add:**
```
go get github.com/jackc/pgx/v5
go get github.com/redis/go-redis/v9
go get github.com/golang-migrate/migrate/v4
```

**Go concepts for this phase:** third-party packages, `pgxpool.New`, `context.Background()`

**Milestone:** `docker compose up -d` → Postgres and Redis are healthy. `go run ./cmd/server` connects to both and prints "connected" with no errors.

---

## Phase 3 — Auth Service (Backend)

**Goal:** SSH key fingerprint → user lookup and registration. Pure Go, no HTTP or gRPC yet.

### Files to create
```
klyntar-server/internal/auth/
├── service.go
└── service_test.go
```

### What `auth.Service` needs to do

```go
// Look up a user by SSH key fingerprint.
// Returns nil, nil if the key is not registered yet.
GetByFingerprint(ctx context.Context, fingerprint string) (*User, error)

// Register a new user and associate their SSH key.
Register(ctx context.Context, email, username, fingerprint, deviceMAC string) (*User, error)

// Add a second SSH key for an existing user (new device).
AddKey(ctx context.Context, userID, fingerprint, deviceName string) error
```

**SSH key fingerprint:** When a user SSHs in, Charm's `wish` library gives you their public key. You call `ssh.FingerprintSHA256(key)` to get a short string like `SHA256:abc123...` — that's your device identity.

**Go concepts for this phase:** structs with pointer receiver methods, `pgx.ErrNoRows` (for "not found"), parameterized SQL queries, writing your first tests with `testing.T`

**Milestone:** A `service_test.go` that registers a user and retrieves them by fingerprint — passing against a real local Postgres.

---

## Phase 4 — Core Domain Services (Backend)

**Goal:** Business logic for posts, voids, comments, and votes. Same struct pattern as auth.

### Files to create
```
klyntar-server/internal/
├── users/service.go        ← GetProfile, UpdateUsername
├── voids/service.go        ← Create, Get, Join, Leave, List, ListByMember
├── posts/service.go        ← Create, Get, List (by void), Delete
├── comments/service.go     ← Create, List (by post, threaded), Delete
└── votes/service.go        ← Upsert (handles upvote, downvote, and removal)
```

**Every service follows this exact pattern:**
```go
type Service struct {
    db    *pgxpool.Pool
    cache *redis.Client  // only include if this service uses Redis
}

func NewService(db *pgxpool.Pool) *Service {
    return &Service{db: db}
}
```

**Tricky part — votes:** A vote is an upsert. If the user votes the same direction twice, remove their vote (toggle). Use `INSERT ... ON CONFLICT DO UPDATE` in SQL.

**Tricky part — nested comments:** `comments` has a `parent_id` column. To load a full comment tree, use a recursive CTE (`WITH RECURSIVE`) in Postgres.

**Go concepts for this phase:** multiple return values, scanning `pgx` rows into structs, DB transactions (`pgx.BeginTx`), table-driven tests

**Milestone:** One test per service that exercises create + read. All passing.

---

## Phase 5 — Feed & Search Services (Backend)

**Goal:** Aggregate queries across tables. Introduce Redis caching.

### Files to create
```
klyntar-server/internal/
├── feed/service.go
└── search/service.go
```

### Feed
`GetFeed(ctx, userID)` returns the top 50 posts from all voids the user has joined, ordered by score descending:

```sql
SELECT p.*
FROM posts p
JOIN void_members vm ON p.void_id = vm.void_id
WHERE vm.user_id = $1
ORDER BY p.score DESC
LIMIT 50
```

**Cache this in Redis** with key `feed:<userID>` and a 60-second TTL. Invalidate the key whenever the user votes or a new post is created in one of their voids.

### Search
`Search(ctx, query)` uses Postgres full-text search:

```sql
SELECT * FROM posts
WHERE to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1)
ORDER BY score DESC LIMIT 20
```

**Go concepts for this phase:** `json.Marshal`/`json.Unmarshal` (for caching structs in Redis), Redis `SET`/`GET`/`DEL` with TTL, Postgres full-text search

**Milestone:** Call `GetFeed` twice — first call hits Postgres, second call hits Redis cache (verify with `redis-cli monitor`).

---

## Phase 6 — Messages Service (Backend)

**Goal:** Direct messages and group rooms. Redis pub/sub for real-time delivery.

### Files to create
```
klyntar-server/internal/messages/
├── service.go   ← CreateRoom, GetRoom, SendMessage, ListMessages (history)
└── broker.go    ← Redis pub/sub wrapper
```

### How real-time works

```
User A sends message
    → service.SendMessage() saves to DB
    → broker.Publish("room:abc123", message)

User B is online
    → broker.Subscribe("room:abc123") returns a channel
    → goroutine reads from channel, streams message to their TUI
```

`broker.go` wraps `redis.PubSub` and gives you a clean Go channel:
```go
func (b *Broker) Subscribe(ctx context.Context, roomID string) (<-chan Message, error)
func (b *Broker) Publish(ctx context.Context, roomID string, msg Message) error
```

**Go concepts for this phase:** goroutines, channels (`chan`, `<-chan`), `context.Done()` for cancellation, `redis.PubSub`

**Milestone:** Two goroutines in a test — one publishes, one subscribes — messages arrive in order with no deadlock.

---

## Phase 7 — gRPC API Layer

**Goal:** Expose all services (Phases 3–6) over gRPC so the TUI can call them.

### Workflow
1. Write `.proto` files describing your API
2. Run `buf generate` → Go code is generated in `gen/pb/`
3. Write a `handler.go` per service that implements the generated interface
4. Wire handlers into `grpc.NewServer()` in `cmd/server/main.go`

### Files to create
```
klyntar-server/
├── api/proto/
│   ├── auth.proto
│   ├── posts.proto
│   ├── voids.proto
│   ├── comments.proto
│   ├── votes.proto
│   ├── feed.proto         ← uses server-streaming RPC
│   ├── messages.proto     ← uses bidirectional streaming RPC
│   └── search.proto
├── buf.yaml
├── buf.gen.yaml
└── internal/
    ├── auth/handler.go
    ├── posts/handler.go
    ├── voids/handler.go
    ├── comments/handler.go
    ├── votes/handler.go
    ├── feed/handler.go
    ├── messages/handler.go
    └── search/handler.go
```

**Proto example (posts.proto):**
```protobuf
syntax = "proto3";
package klyntar.posts.v1;

service PostsService {
  rpc CreatePost(CreatePostRequest) returns (Post);
  rpc GetPost(GetPostRequest) returns (Post);
  rpc ListPosts(ListPostsRequest) returns (ListPostsResponse);
  rpc DeletePost(DeletePostRequest) returns (google.protobuf.Empty);
}
```

**Go concepts for this phase:** Go interfaces (you implement the generated `PostsServiceServer` interface), method embedding, `grpc.NewServer()`, gRPC interceptors (middleware)

**Milestone:** Use `evans` REPL or `grpcurl` to call `ListVoids` → get back an empty array (no crash, no error).

---

## Phase 8 — TUI Foundation (`klyntar/`)

**Goal:** A working SSH server. Any connection gets a placeholder "Welcome" screen. `q` to quit.

### Files to create
```
klyntar/
├── main.go
├── config/config.go
├── internal/
│   ├── server/server.go        ← wish SSH server setup
│   ├── client/grpc.go          ← gRPC client connecting to klyntar-server
│   └── tui/
│       ├── app.go              ← root Bubble Tea model (the "router")
│       └── styles/styles.go    ← lipgloss color palette
└── Dockerfile
```

### How Charm `wish` works

```
User's terminal                    Your server
─────────────                      ─────────────
$ ssh -p 2222 localhost   ──────►  wish.Server accepts connection
                                   Authenticates SSH key
                                   Creates a Bubble Tea program
                                   Streams TUI back to user's terminal ◄───
```

The Bubble Tea program **runs on your server**, not the user's machine. The user's terminal just displays it.

**Go packages to add (klyntar/):**
```
go get github.com/charmbracelet/wish
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
go get google.golang.org/grpc
```

**Bubble Tea's three rules** — every view must implement:
```go
func (m Model) Init() tea.Cmd          // what to do on startup
func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd)  // handle events
func (m Model) View() string           // what to render
```

**Milestone:** `ssh -p 2222 localhost` → "Welcome to klyntar" renders. Press `q` → session closes cleanly.

---

## Phase 9 — TUI Views (`klyntar/`)

**Goal:** Build each screen as its own `tea.Model` with stub/hardcoded data. Wire them into the router.

**Build in this order** (each one teaches a new Bubble Tea pattern):

### 1. Login view — `internal/tui/login/model.go`
Shown when the connecting SSH key isn't registered yet.
- Collect email + username with `bubbles/textinput`
- Submit → call auth service

### 2. Feed view — `internal/tui/feed/model.go`
List of posts from joined voids.
- Use `bubbles/list` component (handles scrolling, filtering for free)
- Each item: title, void name, score, comment count

### 3. Post detail — `internal/tui/post/model.go`
Full post content + comment thread.
- Use `bubbles/viewport` for scrollable content
- `j`/`k` or arrow keys to scroll

### 4. Void browser — `internal/tui/void/model.go`
Browse and join/leave voids.
- Another `bubbles/list`
- `enter` to view, `j` to join, `l` to leave

### 5. Messages — `internal/tui/messages/model.go`
Two-pane: room list on left, message thread on right.
- Left pane: `bubbles/list`
- Right pane: `bubbles/viewport` for history + `bubbles/textarea` for input

### 6. Search — `internal/tui/search/model.go`
Type query → see results.
- `bubbles/textinput` for the query
- Results render in a list below

**Bubble Tea pattern for async data (used in every view):**
```go
// 1. Define a message type for the result
type postsLoadedMsg []Post

// 2. In Init() or on keypress, fire a Cmd
func (m Model) Init() tea.Cmd {
    return func() tea.Msg {
        posts := []Post{{Title: "Stub post"}} // hardcoded for now
        return postsLoadedMsg(posts)
    }
}

// 3. Handle the result in Update()
case postsLoadedMsg:
    m.posts = msg
    return m, nil
```

**Go concepts for this phase:** `switch msg := msg.(type)`, `tea.Batch` (run multiple commands), `lipgloss.JoinVertical/JoinHorizontal` for layout

**Milestone (per view):** SSH in → the view renders with stub data → navigation keys work.

---

## Phase 10 — Wire TUI to gRPC

**Goal:** Replace all stub data with real calls to `klyntar-server`.

For each view, update its `Init()` (or the relevant keypress handler) to call the gRPC client:

```go
// Before (stub):
func loadFeed() tea.Msg {
    return postsLoadedMsg{{Title: "Stub post"}}
}

// After (real):
func (m Model) loadFeed() tea.Msg {
    resp, err := m.grpc.Feed.GetFeed(context.Background(), &pb.GetFeedRequest{
        UserId: m.userID,
    })
    if err != nil {
        return errMsg{err}
    }
    return postsLoadedMsg(resp.Posts)
}
```

**Work through each view:**
- [ ] Login → `auth.Register`
- [ ] Feed → `feed.GetFeed`
- [ ] Post detail → `posts.GetPost` + `comments.ListComments`
- [ ] Void browser → `voids.ListVoids` + `voids.Join` / `voids.Leave`
- [ ] Messages → `messages.StreamMessages` (streaming gRPC)
- [ ] Search → `search.Search`

**Milestone:** Full end-to-end: `ssh` in → register → join a void → create a post → see it in the feed.

---

## Phase 11 — Polish

- [ ] Structured logging with `log/slog` in every service
- [ ] gRPC auth interceptor: verify the caller's user ID on every request
- [ ] Rate limiting on votes and posts (Redis token bucket pattern)
- [ ] `golangci-lint` with no warnings
- [ ] `air` live reload configured for both repos (`.air.toml`)
- [ ] Graceful shutdown: drain in-flight requests before exiting
- [ ] Health check gRPC endpoint on the server

---

## Dependency Map

```
Phase 0 (tools installed)
    └─► Phase 1 (skeleton compiles)
            └─► Phase 2 (Docker + DB schema)
                    └─► Phase 3 (auth service)
                            └─► Phase 4 (posts / voids / comments / votes)
                                    ├─► Phase 5 (feed + search)
                                    └─► Phase 6 (messages + Redis pub/sub)
                                            └─► Phase 7 (gRPC layer)
                                                    └─► Phase 8 (wish SSH server)
                                                            └─► Phase 9 (TUI views with stubs)
                                                                    └─► Phase 10 (wire to real gRPC)
                                                                            └─► Phase 11 (polish)
```

---

## Key Dependencies Reference

### `klyntar-server/go.mod`
```
github.com/jackc/pgx/v5                    ← Postgres driver
github.com/redis/go-redis/v9               ← Redis client
github.com/golang-migrate/migrate/v4       ← DB migrations
google.golang.org/grpc                     ← gRPC server
google.golang.org/protobuf                 ← Protobuf types
```

### `klyntar/go.mod`
```
github.com/charmbracelet/wish              ← SSH server
github.com/charmbracelet/bubbletea         ← TUI framework
github.com/charmbracelet/lipgloss          ← Terminal styling
github.com/charmbracelet/bubbles           ← Pre-built TUI components
google.golang.org/grpc                     ← gRPC client
```
