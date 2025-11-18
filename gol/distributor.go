package gol

import (
	"fmt"
	"net"
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
	DistPort                       string
}

type WorldState struct {
	Turn  int
	World [][]byte
}

type Client struct {
	p Params
	c distributorChannels
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPress <-chan rune) {
	//Establish connection with broker
	broker, err := rpc.Dial("tcp", "localhost:1235")
	if err != nil {
		fmt.Println("Error connecting:", err)
	}
	defer broker.Close()

	//Listen for broker
	fmt.Println("Initialised Client")
	client := new(Client)
	client.p = p
	client.c = c
	clientRpcServer := rpc.NewServer()
	clientRpcServer.Register(client)

	listener, _ := net.Listen("tcp", ":0")
	stopConn := make(chan struct{})
	go func() {
		defer close(stopConn)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go clientRpcServer.ServeConn(conn)
	}()

	fmt.Println("Listening for broker on " + listener.Addr().String())

	//Create the World slice
	world := makeWorld(p.ImageHeight, p.ImageWidth)

	//Fill the World with input
	listOfAlive := []util.Cell{}
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
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

	turn := 0
	pauseFlag := false
	brokerArgs := BrokerArgs{p.ImageHeight, p.ImageWidth, p.Turns, world, listener.Addr().String()}
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
	done := make(chan bool)
	//stopped := make(chan struct{})

	go func() {
		//defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if pauseFlag {
					continue
				}
				var result AliveCellsCount
				_ = broker.Call("Broker.GetAliveCells", brokerArgs, &result)
				turn = result.CompletedTurns
				c.events <- result
			}
		}
	}()

	//TODO: Handle keypresses
	go func() {
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
				currentTurn := turn
				if pauseFlag {
					c.events <- StateChange{currentTurn, Paused}
				} else {
					c.events <- StateChange{currentTurn, Executing}
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
		_ = broker.Call("Broker.GetWorld", true, &state)
		finalWorld = copySliceWorld(state.World)
		fmt.Println("Recieved World, size:", len(finalWorld), "x", len(finalWorld[0]))
		fmt.Println("Turn", state.Turn, " | alive", len(calculateAliveCells(p, finalWorld)))
		//done <- true
		closeClient(p, c, state.Turn, done, ticker, listener, broker, stopConn, finalWorld)
	case <-errChan:
		fmt.Println("GOT RESULT")
		if err != nil {
			fmt.Println("Error:", err)
			closeClient(p, c, turn, done, ticker, listener, broker, stopConn, world)
			return
		}
		fmt.Println("Recieved World, size:", len(finalWorld), "x", len(finalWorld[0]))
		//done <- true
		closeClient(p, c, p.Turns, done, ticker, listener, broker, stopConn, finalWorld) //use p.turns as after it is done, the #turns completed = #turns given
	}
}

func closeClient(p Params, c distributorChannels, turn int, tickerDone chan bool, ticker *time.Ticker, listener net.Listener, rpcClient *rpc.Client, stopConn chan struct{}, world [][]byte) {
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//tickerDone <- struct{}{}
	fmt.Println("CLOSING CLIENT")
	fmt.Println("World has", len(calculateAliveCells(p, world)), "alive")
	tickerDone <- true
	ticker.Stop()

	c.events <- FinalTurnComplete{
		turn,
		calculateAliveCells(p, world),
	}

	//Output PGM image
	outputPGMImage(p, c, turn, world)

	listener.Close()
	<-stopConn

	var dummy bool
	_ = rpcClient.Call("Broker.DisconnectClient", true, &dummy)

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

func (client *Client) TurnComplete(args struct {
	Turn               int
	OldWorld, NewWorld [][]byte
}, result *bool) error {
	for y := 0; y < len(args.OldWorld); y++ {
		for x := 0; x < len(args.OldWorld[0]); x++ {
			if args.OldWorld[y][x] != args.NewWorld[y][x] {
				client.c.events <- CellFlipped{Cell: util.Cell{x, y}, CompletedTurns: args.Turn}
			}
		}
	}

	client.c.events <- TurnComplete{CompletedTurns: args.Turn}
	*result = true
	return nil
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
