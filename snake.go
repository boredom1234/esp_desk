package main

import (
	"math/rand"
	"sync"
	"time"
)


const (
	snakeGridWidth  = 32 
	snakeGridHeight = 16 
	snakeCellSize   = 4  
)


type Point struct {
	X int
	Y int
}


type SnakeGame struct {
	snake     []Point 
	food      Point   
	direction Point   
	gameOver  bool    
	score     int     
	mu        sync.Mutex
}


var snakeGame *SnakeGame

func init() {
	rand.Seed(time.Now().UnixNano())
	initSnakeGame()
}


func initSnakeGame() {
	snakeGame = &SnakeGame{
		snake: []Point{
			{X: snakeGridWidth / 2, Y: snakeGridHeight / 2},
			{X: snakeGridWidth/2 - 1, Y: snakeGridHeight / 2},
			{X: snakeGridWidth/2 - 2, Y: snakeGridHeight / 2},
		},
		direction: Point{X: 1, Y: 0}, 
		gameOver:  false,
		score:     0,
	}
	spawnFood()
}


func spawnFood() {
	for {
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
}


func getSnakeDirection() Point {
	head := snakeGame.snake[0]
	food := snakeGame.food
	currentDir := snakeGame.direction

	
	directions := []Point{
		{X: 0, Y: -1}, 
		{X: 0, Y: 1},  
		{X: -1, Y: 0}, 
		{X: 1, Y: 0},  
	}

	
	validDirs := []Point{}
	for _, dir := range directions {
		if dir.X != -currentDir.X || dir.Y != -currentDir.Y {
			validDirs = append(validDirs, dir)
		}
	}

	
	safeDirs := []Point{}
	for _, dir := range validDirs {
		newHead := Point{X: head.X + dir.X, Y: head.Y + dir.Y}
		if isSafe(newHead) {
			safeDirs = append(safeDirs, dir)
		}
	}

	
	if len(safeDirs) == 0 {
		return currentDir
	}

	
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


func isSafe(p Point) bool {
	
	if p.X < 0 || p.X >= snakeGridWidth || p.Y < 0 || p.Y >= snakeGridHeight {
		return false
	}

	
	for i := 0; i < len(snakeGame.snake)-1; i++ {
		if snakeGame.snake[i].X == p.X && snakeGame.snake[i].Y == p.Y {
			return false
		}
	}

	return true
}


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


func updateSnake() {
	snakeGame.mu.Lock()
	defer snakeGame.mu.Unlock()

	if snakeGame.gameOver {
		
		initSnakeGame()
		return
	}

	
	snakeGame.direction = getSnakeDirection()

	
	head := snakeGame.snake[0]
	newHead := Point{
		X: head.X + snakeGame.direction.X,
		Y: head.Y + snakeGame.direction.Y,
	}

	
	if newHead.X < 0 || newHead.X >= snakeGridWidth ||
		newHead.Y < 0 || newHead.Y >= snakeGridHeight {
		snakeGame.gameOver = true
		return
	}

	
	for _, segment := range snakeGame.snake {
		if segment.X == newHead.X && segment.Y == newHead.Y {
			snakeGame.gameOver = true
			return
		}
	}

	
	snakeGame.snake = append([]Point{newHead}, snakeGame.snake...)

	
	if newHead.X == snakeGame.food.X && newHead.Y == snakeGame.food.Y {
		snakeGame.score++
		spawnFood()
		
	} else {
		
		snakeGame.snake = snakeGame.snake[:len(snakeGame.snake)-1]
	}
}


func generateSnakeFrame(duration int) Frame {
	snakeGame.mu.Lock()
	defer snakeGame.mu.Unlock()

	
	updateSnakeUnlocked()

	elements := []Element{}

	
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

	
	if showHeaders {
		startY = 14
	}

	
	for i, segment := range snakeGame.snake {
		x := segment.X * snakeCellSize
		y := startY + segment.Y*snakeCellSize

		
		if y+snakeCellSize > 64 {
			continue
		}

		
		_ = i 

		
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


func updateSnakeUnlocked() {
	if snakeGame.gameOver {
		
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

	
	snakeGame.direction = getSnakeDirection()

	
	head := snakeGame.snake[0]
	newHead := Point{
		X: head.X + snakeGame.direction.X,
		Y: head.Y + snakeGame.direction.Y,
	}

	
	if newHead.X < 0 || newHead.X >= snakeGridWidth ||
		newHead.Y < 0 || newHead.Y >= snakeGridHeight {
		snakeGame.gameOver = true
		return
	}

	
	for _, segment := range snakeGame.snake {
		if segment.X == newHead.X && segment.Y == newHead.Y {
			snakeGame.gameOver = true
			return
		}
	}

	
	snakeGame.snake = append([]Point{newHead}, snakeGame.snake...)

	
	if newHead.X == snakeGame.food.X && newHead.Y == snakeGame.food.Y {
		snakeGame.score++
		spawnFoodUnlocked()
		
	} else {
		
		snakeGame.snake = snakeGame.snake[:len(snakeGame.snake)-1]
	}
}


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
	
	snakeGame.food = Point{X: 0, Y: 0}
}


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
