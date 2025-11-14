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
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}

type WorldState struct {
	Turn  int
	World [][]byte
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPress <-chan rune) {

	//Establish connection with broker
	broker, err := rpc.Dial("tcp", "localhost:1234")
	if err != nil {
		fmt.Println("Error connecting:", err)
	}
	defer broker.Close()

	//Create the World slice
	world := makeWorld(p.ImageHeight, p.ImageWidth)

	//Fill the World with input
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			world[y][x] = <-c.ioInput
		}
	}

	turn := 0
	brokerArgs := BrokerArgs{p.ImageHeight, p.ImageWidth, p.Turns, world}
	c.events <- StateChange{turn, Executing}

	quitChan := make(chan bool)
	errChan := make(chan error)

	var dummy bool
	_ = broker.Call("Broker.InitBroker", brokerArgs, &dummy)

	//TODO: Calculate turns in broker

	var finalWorld [][]byte
	go func() {
		fmt.Println("Calculating States!!")
		err := broker.Call("Broker.CalculateNStates", brokerArgs, &finalWorld)
		//<-stopped
		fmt.Println("Broker returned with error", err)
		errChan <- err
	}()

	//TODO: Implement turn count ticker
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan struct{})
	//stopped := make(chan struct{})

	go func() {
		//defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				var result AliveCellsCount
				_ = broker.Call("Broker.GetAliveCells", brokerArgs, &result)
				turn = result.CompletedTurns
				c.events <- result
			}
		}
	}()

	//TODO: Handle keypresses
	go func() {
		pauseFlag := false
		for key := range keyPress {
			if key == 's' {
				var state WorldState
				_ = broker.Call("Broker.GetWorld", true, &state)
				currentWorld := copySliceWorld(state.World)
				outputPGMImage(p, c, state.Turn, currentWorld)
			} else if key == 'q' {
				//quitFlag = true
				quitChan <- true
			} else if key == 'p' {
				pauseFlag = !pauseFlag
				if pauseFlag {
					c.events <- StateChange{turn, Paused}
				} else {
					c.events <- StateChange{turn, Executing}
				}
			}
			var callback bool
			_ = broker.Call("Broker.RegisterKey", key, &callback)
		}
	}()

	select {
	case <-quitChan:
		fmt.Println("QUIT CHAN")
		var state WorldState
		broker.Call("Broker.GetWorld", true, &state)
		finalWorld = copySliceWorld(state.World)

		fmt.Println("Recieved World, size:", len(finalWorld), "x", len(finalWorld[0]))
		done <- struct{}{}
		closeClient(p, c, state.Turn, done, ticker, finalWorld)
	case <-errChan:
		fmt.Println("GOT RESULT")
		fmt.Println("Recieved World, size:", len(finalWorld), "x", len(finalWorld[0]))
		done <- struct{}{}
		closeClient(p, c, p.Turns, done, ticker, finalWorld) //use p.turns as after it is done, the #turns completed = #turns given
	}
}

func closeClient(p Params, c distributorChannels, turn int, tickerDone chan struct{}, ticker *time.Ticker, world [][]byte) {
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//tickerDone <- struct{}{}
	fmt.Println("CLOSING CLIENT")
	ticker.Stop()

	c.events <- FinalTurnComplete{
		turn,
		calculateAliveCells(p, world),
	}

	//Output PGM image
	outputPGMImage(p, c, turn, world)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func outputPGMImage(p Params, c distributorChannels, turn int, world [][]byte) {
	c.ioCommand <- ioOutput
	imageName := fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn)
	fmt.Println("Image name:", imageName)
	c.ioFilename <- imageName
	//imageOutput := make([]uint8, p.ImageHeight*p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- ImageOutputComplete{
		turn,
		imageName,
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

func copySliceWorld(world [][]byte) [][]byte {
	copyWorld := make([][]byte, len(world))
	for i := range world {
		copyWorld[i] = make([]byte, len(world[i]))
		copy(copyWorld[i], world[i]) // copy row
	}
	return copyWorld
}
