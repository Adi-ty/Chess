# Chess Backend

A real-time chess server built with Go and WebSockets. Players connect via WebSocket, get matched into games, and play using **UCI (Universal Chess Interface)** notation.

## Setup

### Prerequisites

- Go 1.21+

### Installation

```bash
# Clone the repo
git clone https://github.com/Adi-ty/chess.git
cd chess/backend

# Install dependencies
go mod download

# Run the server
go run cmd/server/main.go
```

Server starts at `ws://localhost:8080/ws`

## Architecture

### How It Works

```
┌─────────────┐         WebSocket          ┌─────────────────┐
│   Player 1  │◄──────────────────────────►│                 │
└─────────────┘                            │   GameManager   │
                                           │                 │
┌─────────────┐         WebSocket          │  ┌───────────┐  │
│   Player 2  │◄──────────────────────────►│  │   Game    │  │
└─────────────┘                            │  │  (board)  │  │
                                           │  └───────────┘  │
                                           └─────────────────┘
```

### GameManager

The `GameManager` is a **singleton** that handles:

1. **Connection Management** - Tracks all connected users
2. **Matchmaking** - Pairs players waiting for a game
3. **Message Routing** - Dispatches messages to correct handlers
4. **Concurrency Safety** - Uses mutex for thread-safe operations

```go
type GameManager struct {
    games       map[string]*Game          // gameID -> Game
    playerGames map[*websocket.Conn]*Game // player -> their current game
    pendingUser *websocket.Conn           // Player waiting for opponent
    users       map[*websocket.Conn]bool  // All connected users
    mu          sync.RWMutex              // Concurrency protection
}
```

**Matchmaking Flow:**

```
Player 1 sends "init_game" → pendingUser = Player1
Player 2 sends "init_game" → Match found! Create game, pendingUser = nil
```

### Game

Each `Game` instance manages:

- **Players** - White and Black WebSocket connections
- **Board State** - Chess position via `notnil/chess`
- **Game Status** - In progress, completed, or abandoned
- **Turn Logic** - Validates correct player is moving

```go
type Game struct {
    ID        string
    white     *websocket.Conn
    black     *websocket.Conn
    board     *chess.Game
    status    GameStatus      // in_progress | completed | abandoned
    startTime time.Time
    endTime   time.Time
    mu        sync.RWMutex    // Concurrency protection
}
```

**Turn validation uses the chess library:**

```go
turn := g.board.Position().Turn()  // Returns chess.White or chess.Black
// Compare against player connection to validate
```

## Message Protocol

All messages are JSON over WebSocket.

### Client → Server

| Type        | Payload              | Description              |
| ----------- | -------------------- | ------------------------ |
| `init_game` | none                 | Join matchmaking queue   |
| `move`      | `{ "move": "e2e4" }` | Make a move (UCI format) |

### Server → Client

| Type         | Payload                                       | Description              |
| ------------ | --------------------------------------------- | ------------------------ |
| `game_start` | `{ "color": "white" }`                        | Game started, your color |
| `move`       | `{ "move": "e2e4" }`                          | Opponent made a move     |
| `game_over`  | `{ "outcome": "1-0", "method": "Checkmate" }` | Game ended               |
| `error`      | `{ "message": "..." }`                        | Error occurred           |

## UCI (Universal Chess Interface) Notation

Moves must be in UCI format (source square + destination square):

| Move    | Meaning                  |
| ------- | ------------------------ |
| `e2e4`  | Pawn from e2 to e4       |
| `g1f3`  | Knight from g1 to f3     |
| `f1b5`  | Bishop from f1 to b5     |
| `e1g1`  | Kingside castle (white)  |
| `e1c1`  | Queenside castle (white) |
| `e4d5`  | Pawn on e4 captures d5   |
| `d1f7`  | Queen from d1 to f7      |
| `e7e8q` | Pawn promotes to queen   |

## Testing with Postman

1. Open **two** WebSocket connections to `ws://localhost:8080/ws`
2. Both send `{"type": "init_game"}` to start a game
3. Exchange moves using `{"type": "move", "move": "e2e4"}`

### Example: Fool's Mate (Fastest Checkmate)

Black wins in 4 moves:

| #   | Player | Send                               |
| --- | ------ | ---------------------------------- |
| 1   | White  | `{"type": "init_game"}`            |
| 2   | Black  | `{"type": "init_game"}`            |
| 3   | White  | `{"type": "move", "move": "f2f3"}` |
| 4   | Black  | `{"type": "move", "move": "e7e5"}` |
| 5   | White  | `{"type": "move", "move": "g2g4"}` |
| 6   | Black  | `{"type": "move", "move": "d8h4"}` |

**Result:** Both receive:

```json
{ "type": "game_over", "outcome": "0-1", "method": "Checkmate" }
```

### Example: Scholar's Mate

White wins in 7 moves:

| #   | Player | Send                               |
| --- | ------ | ---------------------------------- |
| 1   | White  | `{"type": "move", "move": "e2e4"}` |
| 2   | Black  | `{"type": "move", "move": "e7e5"}` |
| 3   | White  | `{"type": "move", "move": "f1c4"}` |
| 4   | Black  | `{"type": "move", "move": "b8c6"}` |
| 5   | White  | `{"type": "move", "move": "d1h5"}` |
| 6   | Black  | `{"type": "move", "move": "g8f6"}` |
| 7   | White  | `{"type": "move", "move": "h5f7"}` |

**Result:** Both receive:

```json
{ "type": "game_over", "outcome": "1-0", "method": "Checkmate" }
```

## State Management

### Server-Side State

All game state lives on the server:

```
GameManager (singleton)
    │
    ├── users: map[*websocket.Conn]bool
    │
    ├── pendingUser: *websocket.Conn (waiting player)
    │
    ├── games: map[string]*Game
    │
    └── playerGames: map[*websocket.Conn]*Game
            │
            └── Game
                ├── ID: string
                ├── white: *websocket.Conn
                ├── black: *websocket.Conn
                ├── board: *chess.Game
                └── status: GameStatus
```

### State Flow

```
1. Connect      → Added to users list
2. init_game    → Either wait (pendingUser) or matched (new Game)
3. move         → Validate turn → Update board → Notify opponent
4. game_over    → Notify both players
5. Disconnect   → Removed from users list
```

## Outcome Values

| Outcome   | Meaning    |
| --------- | ---------- |
| `1-0`     | White wins |
| `0-1`     | Black wins |
| `1/2-1/2` | Draw       |
