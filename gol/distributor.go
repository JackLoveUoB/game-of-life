package gol

import (
	"fmt"
    "net/rpc"
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

type Args struct{
    ImageHeight, ImageWidth, Turns int
    World [][]byte
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	//c.ioFilename <- "128x128"
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	//for _, b
	world := makeWorld(imageHeight, imageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", imageWidth, imageHeight)
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			world[y][x] = <-c.ioInput
		}
	}

	turn := 0
	////c.events = <-	FinalTurnComplete{
	////		turn,
	////		calculateAliveCells(p, world),
	////	}
	if p.Turns == 0 {
		turn = 1
	}
	c.events <- StateChange{turn, Executing}
	//c.ioIdle <- true
	// TODO: Execute all turns of the Game of Life.

	for turn := 0; turn < p.Turns; turn++ {
		var newWorld [][]byte
		if p.Threads == 1 {
		    client, err := rpc.Dial("tcp", "50.17.18.251:1234")
		    if(err != nil){fmt.Println("Error connecting:", err)}
		    defer client.Close()

		    args := Args{ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, Turns: p.Turns, World: world}
            var result [][]byte

            _ = client.Call("Worker.CalculateNextState", args, &result)
            world = result
			//world = calculateNextState(p.ImageHeight, p.ImageWidth, p.Turns, world)
		} else {
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

    client, err := rpc.Dial("tcp", "50.17.18.251:1234")
    if(err != nil){fmt.Println("Error connecting:", err)}
    defer client.Close()

    args := Args{ImageHeight: chunkHeight+2, ImageWidth: imageWidth, Turns: p.Turns, World: chunk}
    var next [][]byte

    _ = client.Call("Worker.CalculateNextState", args, &next)

	//next := calculateNextState(chunkHeight+2, imageWidth, p.Turns, chunk)

	trimItDown := next[1 : len(next)-1]
	out <- trimItDown
}

func populateWorld(oldWorld [][]byte, startX, endX, startY, endY int, newWorld [][]byte) [][]byte {
	anotherY := 0
	for y := startY; y < endY; y++ {
		anotherX := 0
		for x := startX; x < endX; x++ {
			newWorld[anotherY][anotherX] = oldWorld[y][x]
			anotherX++
		}
		anotherY++
	}
	return newWorld
}
