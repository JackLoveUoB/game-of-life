package gol

import (
	"fmt"
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
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	//c.ioFilename <- "128x128"
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	world := makeWorld(imageHeight, imageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", imageWidth, imageHeight)
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			world[y][x] = <-c.ioInput
		}
	}

	turn := 0

	c.events <- StateChange{turn, Executing}

    //Start the AliveCellCount Event
    ticker := time.NewTicker(2 * time.Second)
    allTurnsDone := make(chan bool)
    currentTurnDone := make(chan bool)

    go func(){
        for{
            select{
                case <- allTurnsDone:
                    return
                case <- ticker.C:
                    <- currentTurnDone
                    c.events <- AliveCellsCount{turn, len(calculateAliveCells(p, world))}
            }
        }
    }()



    //Start the calculation of new board states
	for turn = 0; turn < p.Turns; turn++ {
		var newWorld [][]byte
		if p.Threads == 1 {
			world = calculateNextState(p.ImageHeight, p.ImageWidth, p.Turns, world)
    } else {
		    //Calculate chunks for the workers
			chunkHeight := p.ImageHeight / p.Threads
			outyChannels := make([]chan [][]byte, p.Threads)
			for i := 0; i < p.Threads; i++ {
				outyChannels[i] = make(chan [][]byte)
			}
			for i := 0; i < p.Threads; i++ {
				startY := i * chunkHeight
				endY := (i + 1) * chunkHeight
				if i == p.Threads-1 {
					endY = p.ImageHeight
				}

				go worker(
					p,
					world,
					startY,
					endY,
					outyChannels[i])
			}
			for i := 0; i < p.Threads; i++ {
				newWorld = append(newWorld, <-outyChannels[i]...)
			}
			world = newWorld
		}
        currentTurnDone <- true
	}
    //stops the alive cells ticker
    allTurnsDone <- true

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		turn,
		calculateAliveCells(p, world),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

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

func calculateNextState(imageHeight, imageWidth, turns int, world [][]byte) [][]byte {
	if turns == 0 {
		return world
	}
	copyWorld := deepCopy2D(world)
	for i := 0; i < len(world); i++ { //each row
		for j := 0; j < len(world[i]); j++ { //each cell
			alive := 0
			adjacentAlive := 0
			if world[i][j] == 255 {
				alive = 1
			}
			for k := -1; k <= 1; k++ {
				for m := -1; m <= 1; m++ {
					if k == 0 && m == 0 {
						continue
					}
					newi := adjustUpDown(i+k, imageHeight)
					newj := adjustLeftRight(j+m, imageWidth)
					if world[newi][newj] == 255 {
						adjacentAlive++
					}
				}
			}
			if alive == 1 {
				if adjacentAlive < 2 || adjacentAlive > 3 {
					copyWorld[i][j] = 0
				} else {
					copyWorld[i][j] = 255
				}
			} else {
				if adjacentAlive == 3 {
					copyWorld[i][j] = 255
				}
			}

		}

	}
	//world = copyWorld
	return copyWorld
}

func newCell(x, y int) util.Cell {
	return util.Cell{x, y}
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	cells := []util.Cell{}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 255 {
				cells = append(cells, newCell(x, y))
			}
		}
	}
	return cells
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}

func worker(p Params, world [][]byte, startY, endY int, out chan<- [][]byte) {
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth
	chunkHeight := endY - startY

	chunk := makeWorld(chunkHeight+2, imageWidth)

	for y := 0; y < chunkHeight+2; y++ {
		sauceY := (startY + y - 1 + imageHeight) % imageHeight
		copy(chunk[y], world[sauceY])
	}
	next := calculateNextState(chunkHeight+2, imageWidth, p.Turns, chunk)
	trimItDown := next[1 : len(next)-1]
	out <- trimItDown
}
