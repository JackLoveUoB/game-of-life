package gol

import (
	"fmt"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPress2 <-chan rune) {

	// TODO: Create a 2D slice to store the world.
	//c.ioFilename <- "128x128"
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	world := makeWorld(imageHeight, imageWidth)
	var mu sync.Mutex
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", imageWidth, imageHeight)

	listOfAlive := []util.Cell{}
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			world[y][x] = <-c.ioInput
			if world[y][x] == 255 {
				listOfAlive = append(listOfAlive, util.Cell{x, y})
			}
		}
	}

	c.events <- CellsFlipped{
		0,
		listOfAlive,
	}
	quitFlag := false
	//saveFlag := false
	pauseFlag := false
	turn := 0
	completedTurns := 0

	c.events <- StateChange{completedTurns, Executing}

	//Start the AliveCellCount Event
	ticker := time.NewTicker(2 * time.Second)
	allTurnsDone := make(chan bool)
	//currentTurnDone := make(chan bool)

	go func() {
		for {
			select {
			case <-allTurnsDone:

				return
			case <-ticker.C:
				mu.Lock()
				turn2 := completedTurns
				alive2 := len(calculateAliveCells(p, world))
				mu.Unlock()
				c.events <- AliveCellsCount{turn2, alive2}
			}
		}
	}()

	go func() {
		for keys := range keyPress2 {
			if keys == 's' {
				mu.Lock()
				worldSave := deepCopy2D(world)
				saveTurn := completedTurns
				mu.Unlock()
				outputPgm(p, c, worldSave, saveTurn)
			}
			if keys == 'q' {
				quitFlag = true
				return
			}
			if keys == 'p' {
				//pauseChannel <- struct{}{}
				pauseFlag = !pauseFlag
				if pauseFlag {
					c.events <- StateChange{completedTurns, Paused}
				} else {
					c.events <- StateChange{completedTurns, Executing}
				}
			}
		}
	}()

	//Start the calculation of new board states
	for turn = 0; turn < p.Turns; turn++ {

		if quitFlag {
			//turn = p.Turns
			break
		}

		if pauseFlag {
			turn -= 1
		} else {
			mu.Lock()
			world = executer(p, c, completedTurns+1, &mu, world)
			completedTurns++
			mu.Unlock()
			c.events <- TurnComplete{
				completedTurns,
			}
		}
	}

	//stops the alive cells ticker
	allTurnsDone <- true

	// TODO: Report the final state using FinalTurnCompleteEvent.

	outputPgm(p, c, world, completedTurns)

	c.events <- FinalTurnComplete{
		completedTurns,
		calculateAliveCells(p, world),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{completedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func adjustLeftRight(a, b int) int {
	return (a + b) % b
}

func adjustUpDown(a, b int) int {
	return (a + b) % b
}

func deepCopy2D(world [][]byte) [][]byte {
	copyWorld := make([][]byte, len(world))
	for i := range world {
		copyWorld[i] = make([]byte, len(world[i]))
		copy(copyWorld[i], world[i]) // copy row
	}
	return copyWorld
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}

func executer(p Params, c distributorChannels, completedTurns int, mu *sync.Mutex, world [][]byte) [][]byte {
	var newWorld [][]byte

	if p.Threads == 1 {
		newWorld = calculateNextState(c, p, p.ImageWidth, p.Turns, completedTurns, 0, world)
	} else {
		//Calculate chunks for the workers
		chunkHeight := p.ImageHeight / p.Threads
		outyChannels := make([]chan [][]byte, p.Threads)
		for i := 0; i < p.Threads; i++ {
			outyChannels[i] = make(chan [][]byte)
		}
		for i := 0; i < p.Threads; i++ {
			startY := i * chunkHeight // chunkHeight
			endY := (i + 1) * chunkHeight
			if i == p.Threads-1 {
				endY = p.ImageHeight
			}

			go worker(
				p,
				c,
				world,
				startY,
				endY,
				completedTurns,
				outyChannels[i])
		}
		for i := 0; i < p.Threads; i++ {
			newWorld = append(newWorld, <-outyChannels[i]...)
		}

	}
	return newWorld
}
