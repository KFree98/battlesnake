package main

// Welcome to
// __________         __    __  .__                               __
// \______   \_____ _/  |__/  |_|  |   ____   ______ ____ _____  |  | __ ____
//  |    |  _/\__  \\   __\   __\  | _/ __ \ /  ___//    \\__  \ |  |/ // __ \
//  |    |   \ / __ \|  |  |  | |  |_\  ___/ \___ \|   |  \/ __ \|    <\  ___/
//  |________/(______/__|  |__| |____/\_____>______>___|__(______/__|__\\_____>
//
// This file can be a nice home for your Battlesnake logic and helper functions.
//
// To get you started we've included code to prevent your Battlesnake from moving backwards.
// For more info see docs.battlesnake.com

import (
	"log"
	"math/rand"
)

// info is called when you create your Battlesnake on play.battlesnake.com
// and controls your Battlesnake's appearance
// TIP: If you open your Battlesnake URL in a browser you should see this data
func info() BattlesnakeInfoResponse {
	log.Println("INFO")

	return BattlesnakeInfoResponse{
		APIVersion: "1",
		Author:     "kfree98", // TODO: Your Battlesnake username
		Color:      "#FF0000", // TODO: Choose color
		Head:       "pixel",   // TODO: Choose head
		Tail:       "pixel",   // TODO: Choose tail
	}
}

// start is called when your Battlesnake begins a game
func start(state GameState) {
	log.Println("GAME START")
}

// end is called when your Battlesnake finishes a game
func end(state GameState) {
	log.Printf("GAME OVER\n\n")
}

// move is called on every turn and returns your next move
// Valid moves are "up", "down", "left", or "right"
// See https://docs.battlesnake.com/api/example-move for available data
func move(state GameState) BattlesnakeMoveResponse {

	// We've included code to prevent your Battlesnake from moving backwards
	myHead := state.You.Body[0] // Coordinates of your head
	myNeck := state.You.Body[1] // Coordinates of your "neck"

	// TODO: Step 1 - Prevent your Battlesnake from moving out of bounds
	boardWidth := state.Board.Width
	boardHeight := state.Board.Height
	// borderCoords := generateBorderCoords(boardWidth, boardHeight)
	safeMoves := getAvailableMoves(myHead, myNeck, boardWidth, boardHeight, state, state.You.Body)
	log.Println("Safe moves", len(safeMoves))
	// TODO: Step 2 - Prevent your Battlesnake from colliding with itself
	// mybody := state.You.Body

	// TODO: Step 3 - Prevent your Battlesnake from colliding with other Battlesnakes
	// opponents := state.Board.Snakes

	// Are there any safe moves left?

	if len(safeMoves) == 0 {
		log.Printf("MOVE %d: No safe moves detected! Moving down\n", state.Turn)
		return BattlesnakeMoveResponse{Move: "down", Shout: "** PaNiK!! **"}
	}

	// Choose a random move from the safe ones
	nextMove := safeMoves[rand.Intn(len(safeMoves))]

	// TODO: Step 4 - Move towards food instead of random, to regain health and survive longer
	// food := state.Board.Food

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

	var safeMoves []Move
	for _, move := range potentialMoves {
		if move.Safe {
			safeMoves = append(safeMoves, move)
		}
	}

	return safeMoves
}
