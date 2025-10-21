package gol

import "uk.ac.bris.cs/gameoflife/util"

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

	c.ioCommand = make(chan ioCommand)

	//for _, b
	world := make([][]byte, imageHeight)
	for i := range world {
		world[i] = make([]byte, imageWidth)
	}
	c.ioCommand <- ioInput
	c.ioFilename <- "128x128"
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			world[y][x] = <-c.ioInput
		}
	}

	turn := 0
	//c.events = <-	FinalTurnComplete{
	//		turn,
	//		calculateAliveCells(p, world),
	//	}
	if p.Turns == 0 {
		turn = 1
	}
	c.events <- StateChange{turn, Executing}
	//c.ioIdle <- true
	// TODO: Execute all turns of the Game of Life.

	for turn := 0; turn < p.Turns; turn++ {
		world = calculateNextState(p, world)
	}

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

func calculateNextState(p Params, world [][]byte) [][]byte {
	if p.Turns == 0 {
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
					newi := adjustUpDown(i+k, p.ImageHeight)
					newj := adjustLeftRight(j+m, p.ImageWidth)
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
