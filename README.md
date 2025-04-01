
# Conway's Game of Life - Multiplayer Edition

A simple, server-driven implementation of Conway's Game of Life with multiplayer support over a network. All game logic runs on the server, and clients connect via WebSocket to render and interact with the game state in real-time. This project is a proof-of-concept to enable gameplay across multiple computers, bringing back the joy of coding with a nostalgic twist.

## Features
- Multiplayer: Play on multiple devices over a network.
- Server-Side Logic: Entire game state managed by the Go server.
- Patterns: Includes classic patterns like Glider, Gosper Glider Gun, and more, triggered via keyboard commands.
- Live Updates: Real-time board updates broadcasted to all connected clients.

## Prerequisites
- [Go](https://golang.org/dl/) (1.24 or later)
- [Air](https://github.com/air-verse/air) (for live reloading during development)


3. **Install Air**
   Install `air` globally for live reloading:
   ```bash
   go install github.com/air-verse/air@latest

## Running with Air
```bash
air init 
air
```

3. **Access the Game**
    - Open a browser on one computer and navigate to `http://localhost:8080/`.
    - Both clients will connect to the same game instance via WebSocket.


## Usage
- **Start a Game**: Visit the URL in your browser to create or join a game instance (game ID is in the URL, e.g., `/game_xxx`).
- **Commands**: Type these in the browser to interact:
    - `slide`: Add a glider.
    - `blink`: Add a blinker.
    - `toad`: Add a toad.
    - `pulse`: Add a pulsar.
    - `gun`: Add a Gosper Glider Gun.
    - `pent`: Add an R-Pentomino.
    - `snark`: Add a Snark.
    - `engine`: Add a 2-Engine Cordership.
    - `hilbert`: Add a David Hilbert Curve (checkerboard).
    - `clear`: Reset the board.
    - `random`: Randomly spawn cells (50% chance).
    - `stop`: Pause the game.
    - `resume`: Resume the game.
    - `color:<red|blue|green|reset>`: Change background color.

## Multiplayer
- Each client connects to the same `gameID` (from the URL or generated on first visit).
- Actions (e.g., spawning patterns) are sent to the server, which updates the shared state and broadcasts it to all clients.
## Notes
- I know this code is not clean and perfect, but it's a fun project to learn and experiment with Go, WebSockets and clean js as I started programming
- Always have fun!
