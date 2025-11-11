package gol

import (
	"fmt"
	"net/rpc"
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

type Args struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
	Keypress                       chan rune
}

type BrokerArgs struct {
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPress <-chan rune) {
	/*
		//Connect to server
		client, err := rpc.Dial("tcp", "localhost:1234")
		if err != nil {
			fmt.Println("Error connecting:", err)
		}
		defer client.Close()

		//Create the world slice
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

		//Define server arguments
		args := Args{ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, Turns: p.Turns, World: world, Keypress: make(chan rune, 10)}
		turn := 0
		c.events <- StateChange{turn, Executing}

		//Start alivecellscount ticker
		ticker := time.NewTicker(2 * time.Second)
		done := make(chan bool)

		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					var result AliveCellsCount
					_ = client.Call("Server.GetAliveCells", args, &result)
					turn = result.CompletedTurns
					c.events <- result
				}
			}
		}()

		go func() {
			for key := range keyPress {
				var callback bool
				client.Call("Server.RegisterKey", key, &callback)
				if key == 's' {
					var currentWorld [][]byte
					client.Call("Server.GetWorld", args, &currentWorld)
					outputPGMImage(p, c, currentWorld)
				} else if key == 'q' {
					var currentWorld [][]byte
					client.Call("Server.GetWorld", args, &currentWorld)
					closeClient(p, c, done, turn, currentWorld)
				} //p and k are handled server side - if the server shuts down the client will anyway because world returns
			}
		}()

		// Send request to calculate GOL
		var result [][]byte
		_ = client.Call("Server.CalculateNStates", args, &result)
		println("World size", len(result))
		world = result

		closeClient(p, c, done, turn, world)
	*/
	//Establish connection with broker
	node, err := rpc.Dial("tcp", "localhost:1234")
	if err != nil {
		fmt.Println("Error connecting:", err)
	}
	defer node.Close()

	//TODO: Get turns from broker

	var world [][]byte
	node.Call("CalculateNStates")

	//TODO: Implement turn count ticker
}

func closeClient(p Params, c distributorChannels, done chan bool, turn int, world [][]byte) {
	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		turn,
		calculateAliveCells(p, world),
	}

	//Output PGM image
	outputPGMImage(p, c, world)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	done <- true
	close(c.events)
}

func outputPGMImage(p Params, c distributorChannels, world [][]byte) {
	c.ioCommand <- ioOutput
	imageName := fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, p.Turns)
	c.ioFilename <- imageName
	//imageOutput := make([]uint8, p.ImageHeight*p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}
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
	if err != nil {
		fmt.Println("Error connecting:", err)
	}
	defer client.Close()

	args := Args{ImageHeight: chunkHeight + 2, ImageWidth: imageWidth, Turns: p.Turns, World: chunk}
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
