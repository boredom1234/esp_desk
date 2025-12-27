package main

import (
	"math/rand"
	"sync"
	"time"
)

// Snake game constants
const (
	snakeGridWidth  = 32 // 128 pixels / 4 pixel cells
	snakeGridHeight = 16 // 64 pixels / 4 pixel cells
	snakeCellSize   = 4  // Each cell is 4x4 pixels
)

// Point represents a position on the game grid
type Point struct {
	X int
	Y int
}

// SnakeGame holds the complete game state
type SnakeGame struct {
	snake     []Point // Snake body, head is at index 0
	food      Point   // Food position
	direction Point   // Current movement direction
	gameOver  bool    // Whether the game has ended
	score     int     // Current score
	mu        sync.Mutex
}

// Global snake game instance
var snakeGame *SnakeGame

func init() {
	rand.Seed(time.Now().UnixNano())
	initSnakeGame()
}

// initSnakeGame resets the game to initial state
func initSnakeGame() {
	snakeGame = &SnakeGame{
		snake: []Point{
			{X: snakeGridWidth / 2, Y: snakeGridHeight / 2},
			{X: snakeGridWidth/2 - 1, Y: snakeGridHeight / 2},
			{X: snakeGridWidth/2 - 2, Y: snakeGridHeight / 2},
		},
		direction: Point{X: 1, Y: 0}, // Start moving right
		gameOver:  false,
		score:     0,
	}
	spawnFood()
}

// spawnFood places food at a random empty position
func spawnFood() {
	for {
		x := rand.Intn(snakeGridWidth)
		y := rand.Intn(snakeGridHeight)

		// Check if position is not occupied by snake
		occupied := false
		for _, segment := range snakeGame.snake {
			if segment.X == x && segment.Y == y {
				occupied = true
				break
			}
		}

		if !occupied {
			snakeGame.food = Point{X: x, Y: y}
			return
		}
	}
}

// getSnakeDirection uses AI to determine the next direction
func getSnakeDirection() Point {
	head := snakeGame.snake[0]
	food := snakeGame.food
	currentDir := snakeGame.direction

	// Possible directions (can't reverse)
	directions := []Point{
		{X: 0, Y: -1}, // Up
		{X: 0, Y: 1},  // Down
		{X: -1, Y: 0}, // Left
		{X: 1, Y: 0},  // Right
	}

	// Filter out reverse direction
	validDirs := []Point{}
	for _, dir := range directions {
		if dir.X != -currentDir.X || dir.Y != -currentDir.Y {
			validDirs = append(validDirs, dir)
		}
	}

	// Find safe directions (won't immediately cause collision)
	safeDirs := []Point{}
	for _, dir := range validDirs {
		newHead := Point{X: head.X + dir.X, Y: head.Y + dir.Y}
		if isSafe(newHead) {
			safeDirs = append(safeDirs, dir)
		}
	}

	// If no safe directions, just continue (will die)
	if len(safeDirs) == 0 {
		return currentDir
	}

	// Among safe directions, prefer one that moves toward food
	bestDir := safeDirs[0]
	bestDistance := manhattan(Point{X: head.X + bestDir.X, Y: head.Y + bestDir.Y}, food)

	for _, dir := range safeDirs {
		newHead := Point{X: head.X + dir.X, Y: head.Y + dir.Y}
		dist := manhattan(newHead, food)
		if dist < bestDistance {
			bestDistance = dist
			bestDir = dir
		}
	}

	return bestDir
}

// isSafe checks if a position won't cause immediate death
func isSafe(p Point) bool {
	// Check walls
	if p.X < 0 || p.X >= snakeGridWidth || p.Y < 0 || p.Y >= snakeGridHeight {
		return false
	}

	// Check self-collision (excluding tail which will move)
	for i := 0; i < len(snakeGame.snake)-1; i++ {
		if snakeGame.snake[i].X == p.X && snakeGame.snake[i].Y == p.Y {
			return false
		}
	}

	return true
}

// manhattan calculates Manhattan distance between two points
func manhattan(a, b Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// updateSnake moves the snake and handles game logic
func updateSnake() {
	snakeGame.mu.Lock()
	defer snakeGame.mu.Unlock()

	if snakeGame.gameOver {
		// Reset game after death
		initSnakeGame()
		return
	}

	// Get AI direction
	snakeGame.direction = getSnakeDirection()

	// Calculate new head position
	head := snakeGame.snake[0]
	newHead := Point{
		X: head.X + snakeGame.direction.X,
		Y: head.Y + snakeGame.direction.Y,
	}

	// Check for wall collision
	if newHead.X < 0 || newHead.X >= snakeGridWidth ||
		newHead.Y < 0 || newHead.Y >= snakeGridHeight {
		snakeGame.gameOver = true
		return
	}

	// Check for self collision
	for _, segment := range snakeGame.snake {
		if segment.X == newHead.X && segment.Y == newHead.Y {
			snakeGame.gameOver = true
			return
		}
	}

	// Move snake: add new head
	snakeGame.snake = append([]Point{newHead}, snakeGame.snake...)

	// Check if food eaten
	if newHead.X == snakeGame.food.X && newHead.Y == snakeGame.food.Y {
		snakeGame.score++
		spawnFood()
		// Don't remove tail - snake grows
	} else {
		// Remove tail if no food eaten
		snakeGame.snake = snakeGame.snake[:len(snakeGame.snake)-1]
	}
}

// generateSnakeFrame creates a display frame for the snake game
func generateSnakeFrame(duration int) Frame {
	snakeGame.mu.Lock()
	defer snakeGame.mu.Unlock()

	// Update game state
	updateSnakeUnlocked()

	elements := []Element{}

	// Add header if enabled
	startY := 0
	if showHeaders {
		headerText := "= SNAKE ="
		headerSize := getScaledTextSize(1)
		elements = append(elements,
			Element{Type: "text", X: calcCenteredX(headerText, headerSize), Y: 2, Size: headerSize, Value: headerText},
			Element{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
		)
		startY = 14
	}

	// Adjust startY for header mode
	if showHeaders {
		startY = 14
	}

	// Draw snake body as filled rectangles
	for i, segment := range snakeGame.snake {
		x := segment.X * snakeCellSize
		y := startY + segment.Y*snakeCellSize

		// Make sure we don't draw outside bounds
		if y+snakeCellSize > 64 {
			continue
		}

		// Head is slightly different (could add distinction later)
		_ = i // Reserved for future head styling

		// Draw cell as a filled rectangle (using line elements to fill)
		for row := 0; row < snakeCellSize-1; row++ {
			elements = append(elements, Element{
				Type:   "line",
				X:      x,
				Y:      y + row,
				Width:  snakeCellSize - 1,
				Height: 1,
			})
		}
	}

	// Draw food as filled rectangle
	foodX := snakeGame.food.X * snakeCellSize
	foodY := startY + snakeGame.food.Y*snakeCellSize

	if foodY+snakeCellSize <= 64 {
		for row := 0; row < snakeCellSize-1; row++ {
			elements = append(elements, Element{
				Type:   "line",
				X:      foodX,
				Y:      foodY + row,
				Width:  snakeCellSize - 1,
				Height: 1,
			})
		}
	}

	// Add score display if headers are shown
	if showHeaders {
		scoreText := "Score:" + itoa(snakeGame.score)
		elements = append(elements, Element{
			Type:  "text",
			X:     90,
			Y:     55,
			Size:  1,
			Value: scoreText,
		})
	}

	return Frame{
		Version:  1,
		Duration: duration,
		Clear:    true,
		Elements: elements,
	}
}

// updateSnakeUnlocked updates snake without locking (for use when already locked)
func updateSnakeUnlocked() {
	if snakeGame.gameOver {
		// Reset game after death
		snakeGame.snake = []Point{
			{X: snakeGridWidth / 2, Y: snakeGridHeight / 2},
			{X: snakeGridWidth/2 - 1, Y: snakeGridHeight / 2},
			{X: snakeGridWidth/2 - 2, Y: snakeGridHeight / 2},
		}
		snakeGame.direction = Point{X: 1, Y: 0}
		snakeGame.gameOver = false
		snakeGame.score = 0
		spawnFoodUnlocked()
		return
	}

	// Get AI direction
	snakeGame.direction = getSnakeDirection()

	// Calculate new head position
	head := snakeGame.snake[0]
	newHead := Point{
		X: head.X + snakeGame.direction.X,
		Y: head.Y + snakeGame.direction.Y,
	}

	// Check for wall collision
	if newHead.X < 0 || newHead.X >= snakeGridWidth ||
		newHead.Y < 0 || newHead.Y >= snakeGridHeight {
		snakeGame.gameOver = true
		return
	}

	// Check for self collision
	for _, segment := range snakeGame.snake {
		if segment.X == newHead.X && segment.Y == newHead.Y {
			snakeGame.gameOver = true
			return
		}
	}

	// Move snake: add new head
	snakeGame.snake = append([]Point{newHead}, snakeGame.snake...)

	// Check if food eaten
	if newHead.X == snakeGame.food.X && newHead.Y == snakeGame.food.Y {
		snakeGame.score++
		spawnFoodUnlocked()
		// Don't remove tail - snake grows
	} else {
		// Remove tail if no food eaten
		snakeGame.snake = snakeGame.snake[:len(snakeGame.snake)-1]
	}
}

// spawnFoodUnlocked places food without locking
func spawnFoodUnlocked() {
	for attempts := 0; attempts < 100; attempts++ {
		x := rand.Intn(snakeGridWidth)
		y := rand.Intn(snakeGridHeight)

		occupied := false
		for _, segment := range snakeGame.snake {
			if segment.X == x && segment.Y == y {
				occupied = true
				break
			}
		}

		if !occupied {
			snakeGame.food = Point{X: x, Y: y}
			return
		}
	}
	// Fallback if can't find spot (snake is huge)
	snakeGame.food = Point{X: 0, Y: 0}
}

// itoa converts int to string (simple implementation)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}

	if negative {
		result = "-" + result
	}
	return result
}
