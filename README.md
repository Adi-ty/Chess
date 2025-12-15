# Chess Backend

A real-time chess server built with Go and WebSockets. Players connect via WebSocket, get matched into games, and play using **SAN (Standard Algebraic Notation)** moves.

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

```go
type GameManager struct {
    games       []*Game           // Active games
    pendingUser *websocket.Conn   // Player waiting for opponent
    users       []*websocket.Conn // All connected users
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
- **Turn Logic** - Validates correct player is moving

```go
type Game struct {
    white     *websocket.Conn  // First player to join
    black     *websocket.Conn  // Second player to join
    board     *chess.Game      // Chess engine (handles rules)
    startTime time.Time
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

| Type        | Payload            | Description              |
| ----------- | ------------------ | ------------------------ |
| `init_game` | none               | Join matchmaking queue   |
| `move`      | `{ "move": "e4" }` | Make a move (SAN format) |

### Server → Client

| Type         | Payload                                       | Description              |
| ------------ | --------------------------------------------- | ------------------------ |
| `game_start` | `{ "color": "white" }`                        | Game started, your color |
| `move`       | `{ "move": "e4" }`                            | Opponent made a move     |
| `game_over`  | `{ "outcome": "1-0", "method": "Checkmate" }` | Game ended               |
| `error`      | `{ "message": "..." }`                        | Error occurred           |

## SAN (Standard Algebraic Notation)

Moves must be in SAN format:

| Move    | Meaning                      |
| ------- | ---------------------------- |
| `e4`    | Pawn to e4                   |
| `Nf3`   | Knight to f3                 |
| `Bb5`   | Bishop to b5                 |
| `O-O`   | Kingside castle              |
| `O-O-O` | Queenside castle             |
| `exd5`  | Pawn on e-file captures d5   |
| `Qxf7+` | Queen captures f7, check     |
| `Qxf7#` | Queen captures f7, checkmate |

## Testing with Postman

1. Open **two** WebSocket connections to `ws://localhost:8080/ws`
2. Both send `{"type": "init_game"}` to start a game
3. Exchange moves using `{"type": "move", "move": "e4"}`

### Example: Fool's Mate (Fastest Checkmate)

Black wins in 4 moves:

| #   | Player | Send                               |
| --- | ------ | ---------------------------------- |
| 1   | White  | `{"type": "init_game"}`            |
| 2   | Black  | `{"type": "init_game"}`            |
| 3   | White  | `{"type": "move", "move": "f3"}`   |
| 4   | Black  | `{"type": "move", "move": "e5"}`   |
| 5   | White  | `{"type": "move", "move": "g4"}`   |
| 6   | Black  | `{"type": "move", "move": "Qh4#"}` |

**Result:** Both receive:

```json
{ "type": "game_over", "outcome": "0-1", "method": "Checkmate" }
```

### Example: Scholar's Mate

White wins in 7 moves:

| #   | Player | Send                                |
| --- | ------ | ----------------------------------- |
| 1   | White  | `{"type": "move", "move": "e4"}`    |
| 2   | Black  | `{"type": "move", "move": "e5"}`    |
| 3   | White  | `{"type": "move", "move": "Bc4"}`   |
| 4   | Black  | `{"type": "move", "move": "Nc6"}`   |
| 5   | White  | `{"type": "move", "move": "Qh5"}`   |
| 6   | Black  | `{"type": "move", "move": "Nf6"}`   |
| 7   | White  | `{"type": "move", "move": "Qxf7#"}` |

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
    ├── pendingUser: *websocket.Conn (waiting player)
    │
    └── games: []*Game
            │
            └── Game
                ├── white: *websocket.Conn
                ├── black: *websocket.Conn
                └── board: *chess.Game (full chess state)
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
