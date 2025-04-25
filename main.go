package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	gameBoards   = make(map[string]*GameBoard)
	gameBoardsMu sync.RWMutex // Thread safety
)

var debug = true

func trace(name string) func() {
	if !debug {
		return func() {}
	}
	start := time.Now()
	return func() {
		log.Printf("%s took %s", name, time.Since(start))
	}
}

// -- API Responses --------------------------------------------------

func info() BattlesnakeInfoResponse {
	return BattlesnakeInfoResponse{
		APIVersion: "1",
		Author:     "kfree98",
		Color:      "#FF0000",
		Head:       "pixel-round",
		Tail:       "pixel-round",
	}
}

func start(state GameState) {
	log.Println("##############################################################")
	log.Println("GAME START")
	log.Printf("Game ID: %s", state.Game.ID)
	log.Println("Game Ruleset:", state.Game.Ruleset.Name)
	log.Println("##############################################################")

	newBoard := NewGameBoard(state.Game.ID, state.You.ID, state.Board.Width, state.Board.Height)

	AddGameBoard(newBoard)

}

func end(state GameState) {
	log.Printf("GAME OVER\n\n")
	RemoveGameBoard(state.Game.ID)
}

// -- Game Board Lookup ---------------------------------------------

func GetGameBoard(gameID, snakeID string) string {
	defer trace("GetGameBoard")()
	gameBoardsMu.RLock()
	defer gameBoardsMu.RUnlock()

	for _, board := range gameBoards {
		if board.GameID == gameID && board.SnakeID == snakeID {
			return board.ID
		}
	}
	return ""
}

func AddGameBoard(gameBoard *GameBoard) {
	defer trace("AddGameBoard")()
	gameBoardsMu.Lock()
	defer gameBoardsMu.Unlock()
	gameBoards[gameBoard.ID] = gameBoard
}

func RemoveGameBoard(id string) error {
	defer trace("RemoveGameBoard")()
	gameBoardsMu.Lock()
	defer gameBoardsMu.Unlock()

	if _, exists := gameBoards[id]; !exists {
		return fmt.Errorf("game board with ID %s not found", id)
	}
	delete(gameBoards, id)
	return nil
}

// -- Main Handler --------------------------------------------------

func move(state GameState) BattlesnakeMoveResponse {
	defer trace("move")()
	gameBoardsMu.RLock() // Look up the board under read-lock
	gameBoardID := GetGameBoard(state.Game.ID, state.You.ID)
	gameBoardsMu.RUnlock()

	// Get the max execution time
	maxExecutionTime := state.Game.Timeout

	// Get last moves latency
	latency, err := strconv.Atoi(state.You.Latency)
	if err != nil {
		log.Printf("Error converting latency to int: %v", err)
		latency = 0 // Default to 0 if conversion fails
	}
	lastMoveLatency := latency

	thisTurnExecutionTime := maxExecutionTime - ((lastMoveLatency * 2) + 10) // double the latency and add 10ms for safety
	log.Printf("Max execution time: %d", thisTurnExecutionTime)

	// Update board state
	UpdateGameBoard(gameBoardID, state)

	// Get the game board
	currentBoard := gameBoards[gameBoardID]
	currentBoard.DrawBoard()

	// Get the safe instant moves
	safeMoves := currentBoard.GetSafeMoves(state.Board.Snakes, state.You.ID)

	if len(safeMoves) == 0 {
		log.Println("No safe moves available")
		return BattlesnakeMoveResponse{Move: "up", Shout: "Oh no..."}
	}

	randomMove := rand.Intn(len(safeMoves))

	return BattlesnakeMoveResponse{Move: safeMoves[randomMove].Direction}

}

func main() {
	RunServer()
}

// -- Data Structures & Board Setup ---------------------------------

func NewGameBoard(gameID, snakeID string, width, height int) *GameBoard {
	defer trace("NewGameBoard")()

	// Create a new game board with the given dimensions
	grid := make([][]Node, height)
	for y := 0; y < height; y++ {
		// Initialize each row of the grid
		grid[y] = make([]Node, width)
		// Initialize each node in the row
		for x := 0; x < width; x++ {

			grid[y][x] = Node{
				X:       x,
				Y:       y,
				Content: 1, // lower is better. 0 is goal, 1 is clear path, 2 is danger, 3 is never path
			}

		}
	}

	return &GameBoard{
		ID:      uuid.New().String(),
		GameID:  gameID,
		SnakeID: snakeID,
		Width:   width,
		Height:  height,
		Grid:    grid,
	}
}

// Update the board’s danger values
func UpdateGameBoard(id string, state GameState) error {
	defer trace("UpdateGameBoard")()

	gameBoardsMu.RLock()
	board, exists := gameBoards[id]
	gameBoardsMu.RUnlock()

	if !exists {
		return fmt.Errorf("game board with ID %s not found", id)
	}

	// Reset the grid’s Danger
	var wg sync.WaitGroup
	wg.Add(board.Height)
	for y := 0; y < board.Height; y++ {
		go func(y int) {
			defer wg.Done()
			for x := range board.Grid[y] {
				board.Grid[y][x].Content = 1
			}
		}(y)
	}
	wg.Wait()

	// Mark snake positions
	for _, s := range state.Board.Snakes {
		wg.Add(1)
		go func(snake Battlesnake) {
			defer wg.Done()

			if snake.Health == 0 {
				return
			}

			for snakeBodyIndex, coord := range snake.Body {
				node := &board.Grid[coord.Y][coord.X]
				// Mark the snake body
				if snakeBodyIndex == 0 {
					// Head
					node.Content = 4 // Snake head
				} else {
					node.Content = 3 // Snake body
				}
			}
		}(s)
	}
	wg.Wait()

	// Mark hazards
	wg.Add(len(state.Board.Hazards))
	for _, hazard := range state.Board.Hazards {
		go func(hazard Coord) {
			defer wg.Done()
			board.Grid[hazard.Y][hazard.X].Content = 2 // Arena Hazard
		}(hazard)
	}
	wg.Wait()

	// Mark food
	wg.Add(len(state.Board.Food))
	for _, food := range state.Board.Food {
		go func(food Coord) {
			defer wg.Done()
			board.Grid[food.Y][food.X].Content = 0 // Food
		}(food)
	}

	return nil
}

func (board *GameBoard) DrawBoard() {
	for y := board.Height - 1; y >= 0; y-- {
		for x := 0; x < board.Width; x++ {
			switch board.Grid[y][x].Content {
			case 0:
				fmt.Print("G") // Food
			case 1:
				fmt.Print(".") // Clear path
			case 2:
				fmt.Print("^") // Hazard
			case 3:
				fmt.Print("@") // Snake body
			case 4:
				fmt.Print("!") // Snake head
			default:
				fmt.Print("?")
			}
		}
		fmt.Println()
	}
}

// Get the safe moves for a snake. uses the snake ID
func (board *GameBoard) GetSafeMoves(snakes []Battlesnake, snakeId string) []Moves {
	defer trace("GetSafeMoves")()

	// Find the snake in the list of snakes
	var snakeBody []Coord

	for _, snake := range snakes {
		if snake.ID == snakeId {
			snakeBody = snake.Body
			break
		}
	}

	head := snakeBody[0]

	safeMoves := []Moves{
		{Direction: "up", X: head.X, Y: head.Y + 1},
		{Direction: "down", X: head.X, Y: head.Y - 1},
		{Direction: "left", X: head.X - 1, Y: head.Y},
		{Direction: "right", X: head.X + 1, Y: head.Y},
	}

	// Check if any direction collides with any snake's body
	for _, snakes := range snakes {

		if snakes.Health == 0 {
			continue // Skip dead snakes
		}
		for _, coord := range snakes.Body {
			for k, move := range safeMoves {
				if move.X == coord.X && move.Y == coord.Y {
					safeMoves = removeUnordered(safeMoves, k) // Eliminate this move
				}
				if move.X < 0 || move.X >= board.Width || move.Y < 0 || move.Y >= board.Height {
					safeMoves = removeUnordered(safeMoves, k) // Eliminate this move
				}
			}
		}
	}

	return safeMoves
}

func directionTo(from, to Coord) string {
	if to.X > from.X {
		return "right"
	}
	if to.X < from.X {
		return "left"
	}
	if to.Y > from.Y {
		return "up"
	}
	if to.Y < from.Y {
		return "down"
	}
	return "none"

}

// -- Helper Functions ---------------------------------------------
func removeUnordered[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
