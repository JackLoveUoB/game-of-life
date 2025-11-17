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
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

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

	if p.Turns == 0 {
		goto endSequence //skip all instantiation if p.Turns is 0
	}

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
		for key := range keyPresses {
			mu.Lock()
			if key == 's' {
				worldSave := copySliceWorld(world)
				saveTurn := completedTurns
				mu.Unlock()
				outputPgm(p, c, worldSave, saveTurn)
				continue
			}
			if key == 'q' {
				quitFlag = true
				mu.Unlock()
				return
			}
			if key == 'p' {
				//pauseChannel <- struct{}{}
				pauseFlag = !pauseFlag
				turn2 := completedTurns
				newState := Paused
				if !pauseFlag {
					newState = Executing
				}
				mu.Unlock()

				c.events <- StateChange{turn2, newState}
				continue
			}
			mu.Unlock()
		}
		mu.Lock()
		quitFlag = true
		pauseFlag = false
		mu.Unlock()
	}()

	//Start the calculation of new board states
	for turn = 0; turn < p.Turns; turn++ {

		mu.Lock()
		antiRacingQuit := quitFlag
		antiRacingPause := pauseFlag
		mu.Unlock()

		if antiRacingQuit {
			//turn = p.Turns
			break
		}

		if !antiRacingPause {
			mu.Lock()
			antiRacingTurn := completedTurns + 1
			antiRacingWorld := world
			mu.Unlock()

			antiRaceNextWorld := executer(p, c, antiRacingTurn, &mu, antiRacingWorld)

			mu.Lock()
			world = antiRaceNextWorld
			completedTurns = antiRacingTurn
			finalAntiRacingTurn := completedTurns
			mu.Unlock()
			c.events <- TurnComplete{
				finalAntiRacingTurn,
			}
		} else {
			turn--
			time.Sleep(1 * time.Millisecond)
		}
	}

	//stops the alive cells ticker
	allTurnsDone <- true

	// TODO: Report the final state using FinalTurnCompleteEvent.
endSequence:

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

//go:noinline
func adjustLeftRight(finalX, width int) int {
	if finalX < 0 {
		return width - 1
	}
	if finalX >= width {
		return 0
	}
	return finalX
}

//go:noinline
func adjustUpDown(finalY, height int) int {
	if finalY < 0 {
		return height - 1
	}
	if finalY >= height {
		return 0
	}
	return finalY
}

func copySliceWorld(world [][]byte) [][]byte {
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
		newWorld = calculateNextState(c, p, p.Turns, completedTurns, 0, world)
	} else {
		//Calculate chunks for the workers
		chunkHeight := p.ImageHeight / p.Threads
		chunkRemain := p.ImageHeight - (chunkHeight * p.Threads)
		outyChannels := make([]chan [][]byte, p.Threads)
		for i := 0; i < p.Threads; i++ {
			outyChannels[i] = make(chan [][]byte)
		}
		startY := 0
		for i := 0; i < p.Threads; i++ {
			workerChunkHeight := chunkHeight
			if chunkRemain > 0 {
				workerChunkHeight++
				chunkRemain--
			}
			endY := startY + workerChunkHeight
			go worker(
				p,
				c,
				world,
				startY,
				endY,
				completedTurns,
				outyChannels[i])
			startY = endY
		}
		for i := 0; i < p.Threads; i++ {
			newWorld = append(newWorld, <-outyChannels[i]...)
		}

	}
	return newWorld
}
