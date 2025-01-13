package main

import (
	"container/heap"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
)

var (
	gameBoards   = make(map[string]*GameBoard)
	gameBoardsMu sync.RWMutex // Thread safety
)

// -- API Responses --------------------------------------------------

func info() BattlesnakeInfoResponse {
	return BattlesnakeInfoResponse{
		APIVersion: "1",
		Author:     "kfree98",
		Color:      "#FF0000",
		Head:       "pixel",
		Tail:       "pixel",
	}
}

func start(state GameState) {
	log.Println("##############################################################")
	log.Println("GAME START")
	log.Printf("Game ID: %s", state.Game.ID)
	log.Println("Game Ruleset:", state.Game.Ruleset.Name)
	log.Println("##############################################################")

	newBoard := NewGameBoard(state.Game.ID, state.You.ID, state.Board.Width, state.Board.Height)
	UpdateGameBoard(state.Game.ID, state)
	AddGameBoard(newBoard)
}

func end(state GameState) {
	log.Printf("GAME OVER\n\n")
	RemoveGameBoard(state.Game.ID)
}

// -- Game Board Lookup ---------------------------------------------

func GetGameBoard(gameID, snakeID string) string {
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
	gameBoardsMu.Lock()
	defer gameBoardsMu.Unlock()
	gameBoards[gameBoard.ID] = gameBoard
}

func RemoveGameBoard(id string) error {
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
	gameBoardsMu.RLock() // Look up the board under read-lock
	gameBoardID := GetGameBoard(state.Game.ID, state.You.ID)
	board, exists := gameBoards[gameBoardID]
	gameBoardsMu.RUnlock()

	if !exists {
		log.Printf("Game board with ID %s not found", gameBoardID)
		return BattlesnakeMoveResponse{Move: "left"}
	}

	// Update board state
	UpdateGameBoard(gameBoardID, state)

	start := state.You.Head

	if state.You.Health < 40 {
		// Low health => try to get food first
		moveDir, foodFound := Move_GetFood(board, start, state.Board.Food)
		if foodFound {
			log.Println("Moving to food")
			return BattlesnakeMoveResponse{Move: moveDir}
		}
		moveDir, attackFound := Move_Attack(board, start, state.You.Length, state.Board.Snakes)
		if attackFound {
			log.Println("Moving to attack")
			return BattlesnakeMoveResponse{Move: moveDir}
		}
		log.Printf("No path to goal found")
		return BattlesnakeMoveResponse{Move: getFallbackMove(board, start)}

	} else {
		// Otherwise => try to attack first
		moveDir, attackFound := Move_Attack(board, start, state.You.Length, state.Board.Snakes)
		if attackFound {
			log.Println("Moving to attack")
			return BattlesnakeMoveResponse{Move: moveDir}
		}
		moveDir, foodFound := Move_GetFood(board, start, state.Board.Food)
		if foodFound {
			log.Println("Moving to food")
			return BattlesnakeMoveResponse{Move: moveDir}
		}
		log.Printf("No path to goal found")
		return BattlesnakeMoveResponse{Move: getFallbackMove(board, start)}
	}
}

func main() {
	RunServer()
}

// -- Data Structures & Board Setup ---------------------------------

func NewGameBoard(gameID, snakeID string, width, height int) *GameBoard {
	grid := make([][]Node, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]Node, width)
		for x := 0; x < width; x++ {
			grid[y][x] = Node{
				X:      x,
				Y:      y,
				Danger: 1,
			}
		}
	}

	// Preallocate visited & costSoFar arrays
	visited := make([][]bool, height)
	costSoFar := make([][]int, height)
	for y := 0; y < height; y++ {
		visited[y] = make([]bool, width)
		costSoFar[y] = make([]int, width)
	}

	return &GameBoard{
		ID:        uuid.New().String(),
		GameID:    gameID,
		SnakeID:   snakeID,
		Width:     width,
		Height:    height,
		Grid:      grid,
		visited:   visited,
		costSoFar: costSoFar,
	}
}

// Update the board’s danger values
func UpdateGameBoard(id string, state GameState) error {
	gameBoardsMu.RLock()
	board, exists := gameBoards[id]
	gameBoardsMu.RUnlock()

	if !exists {
		return fmt.Errorf("game board with ID %s not found", id)
	}

	// Reset the grid’s Danger
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			board.Grid[y][x].Danger = 1
		}
	}

	// Mark snake positions
	for _, snake := range state.Board.Snakes {
		for i, bodyCoord := range snake.Body {
			if bodyCoord.Y < 0 || bodyCoord.Y >= board.Height ||
				bodyCoord.X < 0 || bodyCoord.X >= board.Width {
				continue
			}
			node := &board.Grid[bodyCoord.Y][bodyCoord.X]

			// Head vs Body
			if i > 0 {
				// Body
				node.Danger = 3 // unpathable
			} else if snake.ID != state.You.ID && snake.Length < state.You.Length {
				node.Danger = 0 // smaller enemy head => low Danger => potential target
			} else {
				node.Danger = 2 // your head or bigger enemy => dangerous
			}
		}
	}

	// Mark hazards
	for _, hazard := range state.Board.Hazards {
		if hazard.Y < 0 || hazard.Y >= board.Height ||
			hazard.X < 0 || hazard.X >= board.Width {
			continue
		}
		board.Grid[hazard.Y][hazard.X].Danger = 3
	}

	return nil
}

// -- Pathfinding ----------------------------------------------------
// Directions: up=(0,+1), down=(0,-1), left=(-1,0), right=(+1,0)

func FindSafestPath(board *GameBoard, start, target Coord) ([]Coord, bool) {
	if start == target {
		return []Coord{start}, true
	}

	directions := []Coord{{0, 1}, {0, -1}, {-1, 0}, {1, 0}}
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			board.visited[y][x] = false
			board.costSoFar[y][x] = 1_000_000_000
		}
	}
	board.costSoFar[start.Y][start.X] = board.Grid[start.Y][start.X].Danger

	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &PriorityQueueItem{
		Coord:     start,
		Priority:  board.costSoFar[start.Y][start.X] + heuristic(start, target),
		CostSoFar: board.costSoFar[start.Y][start.X],
	})

	cameFrom := make(map[Coord]Coord, board.Width*board.Height)

	for pq.Len() > 0 {
		current := heap.Pop(pq).(*PriorityQueueItem)
		cx, cy := current.Coord.X, current.Coord.Y

		if current.Coord == target {
			path := reconstructPath(start, current.Coord, cameFrom)
			log.Printf("Path found from %v to %v: %v", start, target, path)
			return path, true
		}

		if board.visited[cy][cx] {
			continue
		}
		board.visited[cy][cx] = true

		for _, d := range directions {
			nx, ny := cx+d.X, cy+d.Y
			if nx < 0 || nx >= board.Width || ny < 0 || ny >= board.Height {
				continue
			}

			if board.Grid[ny][nx].Danger == 3 {
				continue
			}

			newCost := current.CostSoFar + board.Grid[ny][nx].Danger
			if newCost < board.costSoFar[ny][nx] {
				board.costSoFar[ny][nx] = newCost
				cameFrom[Coord{X: nx, Y: ny}] = current.Coord
				heap.Push(pq, &PriorityQueueItem{
					Coord:     Coord{X: nx, Y: ny},
					Priority:  newCost + heuristic(Coord{X: nx, Y: ny}, target),
					CostSoFar: newCost,
				})
			}
		}
	}

	log.Printf("No path found from %v to %v", start, target)
	return nil, false
}

func reconstructPath(start, end Coord, cameFrom map[Coord]Coord) []Coord {
	path := []Coord{}
	for c := end; c != start; c = cameFrom[c] {
		path = append(path, c)
	}
	path = append(path, start)

	// Reverse the slice in-place
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// -- Movement Helpers -----------------------------------------------

func Move_GetFood(board *GameBoard, start Coord, food []Coord) (string, bool) {
	target, err := FindClosestFoodByHeuristic(board, start, food)
	if err != nil {
		return "", false
	}
	nextCoord, found := FindSafestPath(board, start, target)
	if !found || len(nextCoord) == 0 {
		return "", false
	}
	direction := directionTo(start, nextCoord[1])
	return direction, true
}

func Move_Attack(board *GameBoard, start Coord, myLength int, snakes []Battlesnake) (string, bool) {
	target, found := FindClosestSmallerSnake(board, start, myLength, snakes)
	if !found {
		return "", false
	}
	nextCoord, pathFound := FindSafestPath(board, start, target)
	if !pathFound || len(nextCoord) == 0 {
		return "", false
	}
	direction := directionTo(start, nextCoord[1])
	return direction, true
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
	log.Printf("No direction found from %v to %v, defaulting to 'down'", from, to)
	return "down" // Fallback
}

// getFallbackMove tries all four directions and picks the one with the lowest Danger
func getFallbackMove(board *GameBoard, start Coord) string {
	directions := [4]string{"up", "down", "left", "right"}
	// (dx, dy) for the bottom-left origin system:
	deltas := [4][2]int{
		{0, 1},  // up
		{0, -1}, // down
		{-1, 0}, // left
		{1, 0},  // right
	}
	minDanger := int(^uint(0) >> 1)
	bestMove := "left"

	for i, dir := range directions {
		nx := start.X + deltas[i][0]
		ny := start.Y + deltas[i][1]
		if nx >= 0 && nx < board.Width && ny >= 0 && ny < board.Height {
			danger := board.Grid[ny][nx].Danger
			if danger < minDanger {
				minDanger = danger
				bestMove = dir
			}
		}
	}

	log.Println("Fallback move used:", bestMove)
	return bestMove
}

// -- Distance & Heuristic -------------------------------------------

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Manhattan distance from a to b
func heuristic(a, b Coord) int {
	dx := abs(a.X - b.X)
	dy := abs(a.Y - b.Y)
	return dx + dy
}

// -- Food & Snake Helpers -------------------------------------------

func FindClosestFoodByHeuristic(board *GameBoard, start Coord, food []Coord) (Coord, error) {
	closestFood := Coord{}
	minDistance := int(^uint(0) >> 1) // max int

	for _, f := range food {
		dist := heuristic(start, f)
		if dist < minDistance {
			minDistance = dist
			closestFood = f
		}
	}

	if minDistance == int(^uint(0)>>1) {
		log.Println("No Food Found")
		return Coord{}, fmt.Errorf("no food found")
	}
	return closestFood, nil
}

func FindClosestSmallerSnake(board *GameBoard, start Coord, myLength int, snakes []Battlesnake) (Coord, bool) {
	closestSnake := Coord{}
	minDistance := int(^uint(0) >> 1)

	for _, snake := range snakes {
		if snake.Length >= myLength {
			continue
		}
		enemyHead := snake.Body[0]
		dist := heuristic(start, enemyHead)
		if dist < minDistance {
			minDistance = dist
			closestSnake = enemyHead
		}
	}

	if minDistance == int(^uint(0)>>1) {
		return Coord{}, false
	}
	return closestSnake, true
}

// -- PriorityQueue --------------------------------------------------

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].Priority < pq[j].Priority }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x any) {
	*pq = append(*pq, x.(*PriorityQueueItem))
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}
