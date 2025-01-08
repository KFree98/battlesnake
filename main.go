package main

// __________         __    __  .__                               __
// \______   \_____ _/  |__/  |_|  |   ____   ______ ____ _____  |  | __ ____
//  |    |  _/\__  \\   __\   __\  | _/ __ \ /  ___//    \\__  \ |  |/ // __ \
//  |    |   \ / __ \|  |  |  | |  |_\  ___/ \___ \|   |  \/ __ \|    <\  ___/
//  |________/(______/__|  |__| |____/\_____>______>___|__(______/__|__\\_____>
//

import (
	"log"
	"math/rand"
)

// Info is the server ping
func info() BattlesnakeInfoResponse {
	log.Println("INFO")

	return BattlesnakeInfoResponse{
		APIVersion: "1",
		Author:     "kfree98",
		Color:      "#FF0000",
		Head:       "pixel",
		Tail:       "pixel",
	}
}

// start is called when your Battlesnake begins a game
func start(state GameState) {
	log.Println("##############################################################")
	log.Println("GAME START")
	log.Printf("Game ID: %s", state.Game.ID)
	log.Println("Game Ruleset:", state.Game.Ruleset.Name)
	log.Println("##############################################################")
}

// end is called when your Battlesnake finishes a game
func end(state GameState) {
	log.Printf("GAME OVER\n\n")
}

func move(state GameState) BattlesnakeMoveResponse {

	// log.Println(state.Board.Snakes)

	myHead := state.You.Body[0]
	myNeck := state.You.Body[1]

	boardWidth := state.Board.Width
	boardHeight := state.Board.Height

	// Calculate the available moves
	SafeMoves := getMySafeMoves(myHead, myNeck, boardWidth, boardHeight, state, state.You.Body)
	EnemyMoves := getEnemyPotentialMoves(state)

	FinalSafeMoves := AvoidHeadToHeads(SafeMoves, EnemyMoves, state)

	log.Println("Safe moves", len(FinalSafeMoves))

	// If no safe moves, lay down in defeat..
	if len(FinalSafeMoves) == 0 {
		log.Printf("MOVE %d: No safe moves detected! Moving down\n", state.Turn)
		return BattlesnakeMoveResponse{Move: "down", Shout: "** PaNiK!! **"}
	}

	// Choose a random move from the safe ones
	nextMove := FinalSafeMoves[rand.Intn(len(FinalSafeMoves))]

	log.Printf("Snake: %s  MOVE %d: %s\n", state.You.Name, state.Turn, nextMove.Direction)
	return BattlesnakeMoveResponse{Move: nextMove.Direction}
}

func main() {
	RunServer()
}

func getEnemyPotentialMoves(state GameState) []EnemyMoves {
	var enemyMoves []EnemyMoves

	for _, snake := range state.Board.Snakes {
		// Skip your own snake
		if snake.ID == state.You.ID {
			continue
		}

		// Ensure the snake has at least 2 body parts to calculate moves
		if len(snake.Body) < 2 {
			continue
		}

		head := snake.Body[0]
		neck := snake.Body[1]

		// Get potential moves for this enemy snake
		potentialMoves := getPotentialMoveCoords(head, neck)

		// Collect the coordinates of potential moves
		var coords []Coord
		for _, move := range potentialMoves {
			if move.Safe {
				coords = append(coords, move.Coord)
			}
		}

		// Append to the enemy moves list
		enemyMoves = append(enemyMoves, EnemyMoves{Coords: coords})
	}

	return enemyMoves
}

// Returns the potential Moves for a given head and neck
func getPotentialMoveCoords(head, neck Coord) []Move {
	potentialMoves := []Move{
		{Coord: Coord{X: head.X + 1, Y: head.Y}, Direction: "right", Safe: true},
		{Coord: Coord{X: head.X - 1, Y: head.Y}, Direction: "left", Safe: true},
		{Coord: Coord{X: head.X, Y: head.Y + 1}, Direction: "up", Safe: true},
		{Coord: Coord{X: head.X, Y: head.Y - 1}, Direction: "down", Safe: true},
	}

	// Mark moves that would go back to the neck as unsafe
	for i := range potentialMoves {
		move := &potentialMoves[i]
		if move.Coord == neck {
			move.Safe = false
			break
		}
	}
	// Return Safe moves for the head
	return potentialMoves
}

func AvoidHeadToHeads(safeMoves []Move, enemyMoves []EnemyMoves, state GameState) []Move {
	// Create a map to store all enemy move coordinates for fast lookup
	enemyCoordsMap := make(map[Coord]bool)

	// Populate the map with enemy move coordinates
	for _, enemy := range enemyMoves {
		for _, coord := range enemy.Coords {
			enemyCoordsMap[coord] = true
		}
	}

	// Iterate through your safe moves and mark conflicts as unsafe
	for i := range safeMoves {
		if enemyCoordsMap[safeMoves[i].Coord] {
			safeMoves[i].Safe = false
			log.Println("Snake: %s Avoiding head-to-head collision for move", state.You.Name, safeMoves[i])
		}
	}

	var finalSafeMoves []Move
	for _, move := range safeMoves {
		if move.Safe {
			finalSafeMoves = append(finalSafeMoves, move)
		}
	}

	return finalSafeMoves
}

func getMySafeMoves(head, neck Coord, boardWidth, boardHeight int, state GameState, body []Coord) []Move {

	// Define all potential moves with directions
	potentialMoves := getPotentialMoveCoords(head, neck)

	// Mark moves that would go back to the neck as unsafe
	for i := range potentialMoves {
		move := &potentialMoves[i]

		// Check if move is outside the grid boundaries
		if move.Coord.X < 0 || move.Coord.X >= boardWidth || move.Coord.Y < 0 || move.Coord.Y >= boardHeight {
			move.Safe = false
			continue
		}

		// Check if move collides with any part of the body
		for _, segment := range body {
			if move.Coord == segment {
				move.Safe = false
				break
			}
		}

		// Check if move collides with any other snake
		for _, snake := range state.Board.Snakes {
			for _, segment := range snake.Body {
				if move.Coord == segment {
					move.Safe = false
					log.Println("Snake collision for move", move)
					continue
				}
			}
		}

	}

	// Filter out unsafe moves
	var safeMoves []Move
	for _, move := range potentialMoves {
		if move.Safe {
			safeMoves = append(safeMoves, move)
		}
	}

	// Return the safe moves
	return safeMoves
}
