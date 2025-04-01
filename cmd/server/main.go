package main

import (
	"encoding/base64"
	"log"
	"math/rand"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GameState struct {
	Board           [][]uint8
	Width           int
	Height          int
	CellSize        int
	Color           string
	BackgroundColor string
	Interval        int64
	Stopped         bool
	mu              sync.Mutex
}

type Client struct {
	conn   *websocket.Conn
	gameID string
}

var (
	games    = make(map[string]*GameState)
	clients  = make(map[*Client]bool)
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	mutex = sync.Mutex{}
)

func NewGameState(width, height, cellSize int, color, bgColor string, interval int64) *GameState {
	board := make([][]uint8, height)
	for i := range board {
		board[i] = make([]uint8, width)
	}
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			if rand.Intn(100) < 20 {
				board[y][x] = 100
				board[y-1][x]++
				board[y+1][x]++
				board[y-1][x-1]++
				board[y-1][x+1]++
				board[y][x-1]++
				board[y][x+1]++
				board[y+1][x-1]++
				board[y+1][x+1]++
			}
		}
	}
	return &GameState{
		Board:           board,
		Width:           width,
		Height:          height,
		CellSize:        cellSize,
		Color:           color,
		BackgroundColor: bgColor,
		Interval:        interval,
	}
}

func (g *GameState) Birth(x, y int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if x > 0 && x < g.Width-1 && y > 0 && y < g.Height-1 && g.Board[y][x] < 100 {
		g.Board[y][x] = 100
		g.Board[y-1][x]++
		g.Board[y+1][x]++
		g.Board[y-1][x-1]++
		g.Board[y-1][x+1]++
		g.Board[y][x-1]++
		g.Board[y][x+1]++
		g.Board[y+1][x-1]++
		g.Board[y+1][x+1]++
		log.Printf("[Game] Birth at (x: %d, y: %d)", x, y)
	} else {
		log.Printf("[Game] Birth failed at (x: %d, y: %d) - out of bounds or already alive", x, y)
	}
}

func (g *GameState) Update() {
	g.mu.Lock()
	defer g.mu.Unlock()
	newBoard := make([][]uint8, g.Height)
	for i := range newBoard {
		newBoard[i] = make([]uint8, g.Width)
	}

	liveCells := 0
	for y := 1; y < g.Height-1; y++ {
		for x := 1; x < g.Width-1; x++ {
			neighbors := g.Board[y][x]
			if neighbors >= 100 {
				neighbors -= 100
			}
			isAlive := g.Board[y][x] >= 100
			if (isAlive && (neighbors == 2 || neighbors == 3)) || (!isAlive && neighbors == 3) {
				newBoard[y][x] = 100
				newBoard[y-1][x]++
				newBoard[y+1][x]++
				newBoard[y-1][x-1]++
				newBoard[y-1][x+1]++
				newBoard[y][x-1]++
				newBoard[y][x+1]++
				newBoard[y+1][x-1]++
				newBoard[y+1][x+1]++
				liveCells++
			}
		}
	}
	g.Board = newBoard
	if liveCells == 0 {
		g.Stopped = true
		log.Printf("[Game] Game stopped - No live cells remaining")
	}
	log.Printf("[Game] Updated game state - Live cells: %d, Stopped: %v", liveCells, g.Stopped)
}

func gameLoop() {
	for {
		mutex.Lock()
		gameIDs := make([]string, 0, len(games))
		for gameID := range games {
			gameIDs = append(gameIDs, gameID)
		}
		mutex.Unlock()

		for _, gameID := range gameIDs {
			mutex.Lock()
			game, exists := games[gameID]
			mutex.Unlock()
			if !exists {
				continue
			}
			if !game.Stopped {
				game.Update()
				broadcastGameState(game, gameID)
				log.Printf("[GameLoop] Broadcasted state for gameID: %s", gameID)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func broadcastGameState(game *GameState, gameID string) {
	mutex.Lock()
	defer mutex.Unlock()
	encodedBoard := make([]string, len(game.Board))
	for i, row := range game.Board {
		encodedBoard[i] = base64.StdEncoding.EncodeToString(row)
	}
	gameState := struct {
		Board           []string
		Width           int
		Height          int
		CellSize        int
		Color           string
		BackgroundColor string
		Interval        int64
		Stopped         bool
	}{
		Board:           encodedBoard,
		Width:           game.Width,
		Height:          game.Height,
		CellSize:        game.CellSize,
		Color:           game.Color,
		BackgroundColor: game.BackgroundColor,
		Interval:        game.Interval,
		Stopped:         game.Stopped,
	}
	for client := range clients {
		if client.gameID == gameID {
			err := client.conn.WriteJSON(gameState)
			if err != nil {
				log.Printf("[Broadcast] Error sending to client for gameID %s: %v", gameID, err)
				client.conn.Close()
				delete(clients, client)
			} else {
				log.Printf("[Broadcast] Successfully sent game state to client for gameID: %s", gameID)
			}
		}
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	client := &Client{conn: conn, gameID: ""}
	mutex.Lock()
	clients[client] = true
	mutex.Unlock()
	log.Printf("[WebSocket] New client connected (gameID TBD)")

	defer func() {
		mutex.Lock()
		delete(clients, client)
		mutex.Unlock()
		conn.Close()
		log.Printf("[WebSocket] Client disconnected for gameID: %s", client.gameID)
	}()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("[WebSocket] Read error: %v", err)
			return
		}
		log.Printf("[WebSocket] Received message from client: %v", msg)
		handleClientMessage(client, msg)
	}
}

func handleClientMessage(client *Client, msg map[string]interface{}) {
	gameID, ok := msg["gameID"].(string)
	if !ok {
		log.Printf("[Handler] No gameID in message: %v", msg)
		return
	}

	mutex.Lock()
	game, exists := games[gameID]
	if !exists && msg["type"] == "init" {
		width := int(msg["width"].(float64))
		height := int(msg["height"].(float64))
		cellSize := int(msg["cellSize"].(float64))
		game = NewGameState(width, height, cellSize, "#ccc", "#111", 1000000000)
		games[gameID] = game
		client.gameID = gameID
		log.Printf("[Handler] Initialized new game for gameID: %s with dimensions %dx%d", gameID, width, height)
	} else if exists {
		client.gameID = gameID
		log.Printf("[Handler] Client joined existing game for gameID: %s", gameID)
	}
	mutex.Unlock()

	if !exists && msg["type"] != "init" {
		log.Printf("[Handler] GameID %s not found for message: %v", gameID, msg)
		return
	}

	switch msg["type"] {
	case "init":
		broadcastGameState(game, gameID)
	case "birth":
		x := int(msg["x"].(float64))
		y := int(msg["y"].(float64))
		game.Birth(x, y)
		broadcastGameState(game, gameID)
	case "stop":
		game.Stopped = true
		log.Printf("[Handler] Game stopped for gameID: %s", gameID)
		broadcastGameState(game, gameID)
	case "resume":
		game.Stopped = false
		log.Printf("[Handler] Game resumed for gameID: %s", gameID)
		broadcastGameState(game, gameID)
	case "setBackgroundColor":
		color := msg["color"].(string)
		game.BackgroundColor = color
		log.Printf("[Handler] Set background color to %s for gameID: %s", color, gameID)
		broadcastGameState(game, gameID)
	case "clear":
		game.mu.Lock()
		for y := 0; y < game.Height; y++ {
			for x := 0; x < game.Width; x++ {
				game.Board[y][x] = 0
			}
		}
		game.mu.Unlock()
		log.Printf("[Handler] Cleared board for gameID: %s", gameID)
		broadcastGameState(game, gameID)
	case "randomBirth":
		percentage := int(msg["percentage"].(float64))
		log.Printf("[Handler] Starting random birth with %d%% for gameID: %s", percentage, gameID)
		game.mu.Lock()
		defer func() {
			game.mu.Unlock()
			if r := recover(); r != nil {
				log.Printf("[Handler] Panic in randomBirth for gameID %s: %v", gameID, r)
			}
			log.Printf("[Handler] Random birth with %d%% completed for gameID: %s", percentage, gameID)
			broadcastGameState(game, gameID)
		}()
		for y := 1; y < game.Height-1; y++ {
			for x := 1; x < game.Width-1; x++ {
				if game.Board[y][x] < 100 && percentage > rand.Intn(100) {
					if x > 0 && x < game.Width-1 && y > 0 && y < game.Height-1 && game.Board[y][x] < 100 {
						game.Board[y][x] = 100
						game.Board[y-1][x]++
						game.Board[y+1][x]++
						game.Board[y-1][x-1]++
						game.Board[y-1][x+1]++
						game.Board[y][x-1]++
						game.Board[y][x+1]++
						game.Board[y+1][x-1]++
						game.Board[y+1][x+1]++
					}
				}
			}
		}
	case "pattern":
		pattern := msg["pattern"].(string)
		log.Printf("[Handler] Applying pattern %s for gameID: %s", pattern, gameID)
		game.mu.Lock()
		defer func() {
			game.mu.Unlock()
			if r := recover(); r != nil {
				log.Printf("[Handler] Panic in pattern for gameID %s: %v", gameID, r)
			}
			log.Printf("[Handler] Pattern %s applied for gameID: %s", pattern, gameID)
			broadcastGameState(game, gameID)
		}()
		switch pattern {
		case "glider":
			xOffset := rand.Intn(game.Width-5) + 1 // 3x3 pattern
			yOffset := rand.Intn(game.Height-5) + 1
			glider := [][]int{
				{0, 1, 0},
				{0, 0, 1},
				{1, 1, 1},
			}
			for y := 0; y < 3; y++ {
				for x := 0; x < 3; x++ {
					if glider[y][x] == 1 {
						game.Board[yOffset+y][xOffset+x] = 100
						game.Board[yOffset+y-1][xOffset+x]++
						game.Board[yOffset+y+1][xOffset+x]++
						game.Board[yOffset+y-1][xOffset+x-1]++
						game.Board[yOffset+y-1][xOffset+x+1]++
						game.Board[yOffset+y][xOffset+x-1]++
						game.Board[yOffset+y][xOffset+x+1]++
						game.Board[yOffset+y+1][xOffset+x-1]++
						game.Board[yOffset+y+1][xOffset+x+1]++
					}
				}
			}
		case "blinker":
			xOffset := rand.Intn(game.Width-4) + 1 // 3x1 pattern
			yOffset := rand.Intn(game.Height-2) + 1
			blinker := [][]int{
				{1, 1, 1}, // Horizontal line of 3 cells
			}
			for y := 0; y < 1; y++ {
				for x := 0; x < 3; x++ {
					if blinker[y][x] == 1 {
						game.Board[yOffset+y][xOffset+x] = 100
						game.Board[yOffset+y-1][xOffset+x]++
						game.Board[yOffset+y+1][xOffset+x]++
						game.Board[yOffset+y-1][xOffset+x-1]++
						game.Board[yOffset+y-1][xOffset+x+1]++
						game.Board[yOffset+y][xOffset+x-1]++
						game.Board[yOffset+y][xOffset+x+1]++
						game.Board[yOffset+y+1][xOffset+x-1]++
						game.Board[yOffset+y+1][xOffset+x+1]++
					}
				}
			}
		case "toad":
			xOffset := rand.Intn(game.Width-5) + 1 // 4x2 pattern
			yOffset := rand.Intn(game.Height-3) + 1
			toad := [][]int{
				{0, 1, 1, 1},
				{1, 1, 1, 0},
			}
			for y := 0; y < 2; y++ {
				for x := 0; x < 4; x++ {
					if toad[y][x] == 1 {
						game.Board[yOffset+y][xOffset+x] = 100
						game.Board[yOffset+y-1][xOffset+x]++
						game.Board[yOffset+y+1][xOffset+x]++
						game.Board[yOffset+y-1][xOffset+x-1]++
						game.Board[yOffset+y-1][xOffset+x+1]++
						game.Board[yOffset+y][xOffset+x-1]++
						game.Board[yOffset+y][xOffset+x+1]++
						game.Board[yOffset+y+1][xOffset+x-1]++
						game.Board[yOffset+y+1][xOffset+x+1]++
					}
				}
			}
		case "pulsar":
			xOffset := rand.Intn(game.Width-13) + 1 // 13x13 pattern
			yOffset := rand.Intn(game.Height-13) + 1
			pulsar := [][]int{
				{0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1},
				{1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1},
				{1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1},
				{0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0},
				{1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1},
				{1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1},
				{1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0},
			}
			for y := 0; y < 13; y++ {
				for x := 0; x < 13; x++ {
					if pulsar[y][x] == 1 {
						game.Board[yOffset+y][xOffset+x] = 100
						game.Board[yOffset+y-1][xOffset+x]++
						game.Board[yOffset+y+1][xOffset+x]++
						game.Board[yOffset+y-1][xOffset+x-1]++
						game.Board[yOffset+y-1][xOffset+x+1]++
						game.Board[yOffset+y][xOffset+x-1]++
						game.Board[yOffset+y][xOffset+x+1]++
						game.Board[yOffset+y+1][xOffset+x-1]++
						game.Board[yOffset+y+1][xOffset+x+1]++
					}
				}
			}
		case "gosper_glider_gun":
			// Gosper Glider Gun (36x9, scaled to fit ~half the board)
			scale := int(float64(game.Width) / 36 / 2) // Scale to ~half width
			if scale < 1 {
				scale = 1
			}
			xOffset := (game.Width - 36*scale) / 2 // Center horizontally
			yOffset := (game.Height - 9*scale) / 2 // Center vertically
			gun := [][]int{
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1},
				{1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			}
			for y := 0; y < 9; y++ {
				for x := 0; x < 36; x++ {
					if gun[y][x] == 1 {
						for sy := 0; sy < scale; sy++ {
							for sx := 0; sx < scale; sx++ {
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx] = 100
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx+1]++
							}
						}
					}
				}
			}
		case "r_pentomino":
			// R-Pentomino (3x3, scaled to ~half the board)
			scale := int(float64(game.Width) / 3 / 2) // Scale to ~half width
			if scale < 1 {
				scale = 1
			}
			xOffset := (game.Width - 3*scale) / 2
			yOffset := (game.Height - 3*scale) / 2
			rPentomino := [][]int{
				{0, 1, 1},
				{1, 1, 0},
				{0, 1, 0},
			}
			for y := 0; y < 3; y++ {
				for x := 0; x < 3; x++ {
					if rPentomino[y][x] == 1 {
						for sy := 0; sy < scale; sy++ {
							for sx := 0; sx < scale; sx++ {
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx] = 100
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx+1]++
							}
						}
					}
				}
			}
		case "snark":
			// Snark (still life, 34x34, scaled to ~half the board)
			scale := int(float64(game.Width) / 34 / 2)
			if scale < 1 {
				scale = 1
			}
			xOffset := (game.Width - 34*scale) / 2
			yOffset := (game.Height - 34*scale) / 2
			snark := [][]int{
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			}
			for y := 0; y < 34; y++ {
				for x := 0; x < 34; x++ {
					if snark[y][x] == 1 {
						for sy := 0; sy < scale; sy++ {
							for sx := 0; sx < scale; sx++ {
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx] = 100
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx+1]++
							}
						}
					}
				}
			}
		case "2_engine":
			// 2-Engine Cordership (19x19, scaled to ~half the board)
			scale := int(float64(game.Width) / 19 / 2)
			if scale < 1 {
				scale = 1
			}
			xOffset := (game.Width - 19*scale) / 2
			yOffset := (game.Height - 19*scale) / 2
			twoEngine := [][]int{
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 00, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			}
			for y := 0; y < 19; y++ {
				for x := 0; x < 19; x++ {
					if twoEngine[y][x] == 1 {
						for sy := 0; sy < scale; sy++ {
							for sx := 0; sx < scale; sx++ {
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx] = 100
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx+1]++
							}
						}
					}
				}
			}

		case "david_hilbert":
			// David Hilbert Curve (approximated as a large square grid, 64x64, scaled to fit ~75% of board)
			scale := int(float64(game.Width) / 64 * 3 / 4) // Scale to ~75% width
			if scale < 1 {
				scale = 1
			}
			xOffset := (game.Width - 64*scale) / 2
			yOffset := (game.Height - 64*scale) / 2
			// Simplified Hilbert curve as a grid (actual curve requires recursive generation, here we approximate)
			for y := 0; y < 64; y++ {
				for x := 0; x < 64; x++ {
					if (x+y)%2 == 0 { // Checkerboard pattern for visibility
						for sy := 0; sy < scale; sy++ {
							for sx := 0; sx < scale; sx++ {
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx] = 100
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy-1][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy][xOffset+x*scale+sx+1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx-1]++
								game.Board[yOffset+y*scale+sy+1][xOffset+x*scale+sx+1]++
							}
						}
					}
				}
			}
		default:
			log.Printf("[Handler] Unknown pattern %s for gameID: %s", pattern, gameID)
		}
	}
}

// Serve static files and index.html for SPA routes
func serveHandler(w http.ResponseWriter, r *http.Request) {
	// Clean the requested path
	p := path.Clean(r.URL.Path)
	if p == "/" || strings.HasPrefix(p, "/game_") {
		// Serve index.html for root and game IDs
		log.Printf("[HTTP] Serving index.html for path: %s", p)
		http.ServeFile(w, r, "./assets/index.html")
		return
	}

	// Serve static files from assets directory
	fs := http.FileServer(http.Dir("./assets"))
	fs.ServeHTTP(w, r)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/", serveHandler)
	go gameLoop()
	log.Printf("[Main] Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
