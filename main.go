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

	myHead := state.You.Body[0]
	myNeck := state.You.Body[1]

	boardWidth := state.Board.Width
	boardHeight := state.Board.Height

	// Calculate the available moves
	safeMoves := getAvailableMoves(myHead, myNeck, boardWidth, boardHeight, state, state.You.Body)
	log.Println("Safe moves", len(safeMoves))

	// If no safe moves, lay down in defeat..
	if len(safeMoves) == 0 {
		log.Printf("MOVE %d: No safe moves detected! Moving down\n", state.Turn)
		return BattlesnakeMoveResponse{Move: "down", Shout: "** PaNiK!! **"}
	}

	// Choose a random move from the safe ones
	nextMove := safeMoves[rand.Intn(len(safeMoves))]

	log.Printf("MOVE %d: %s\n", state.Turn, nextMove.Direction)
	return BattlesnakeMoveResponse{Move: nextMove.Direction}
}

func main() {
	RunServer()
}

func getAvailableMoves(head, neck Coord, boardWidth, boardHeight int, state GameState, body []Coord) []Move {

	// Define all potential moves with directions
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
			continue
		}
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
