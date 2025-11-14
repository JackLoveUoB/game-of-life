package main

import (
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"time"
)

type Broker struct {
	turn       int
	world      [][]byte
	mu         sync.Mutex
	KeyPresses chan rune
	Listener   net.Listener
}

type BrokerArgs struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}

type NodeArgs struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}

type AliveCellsCount struct {
	CompletedTurns int
	CellsCount     int
}

func main() {
	//TODO: handle connections
	broker := new(Broker)
	rpc.Register(broker)

	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		fmt.Println("Error listening:", err)
	}
	fmt.Println("Listening on port 1234...")

	broker.KeyPresses = make(chan rune, 10)
	broker.Listener = listener

	rpc.Accept(listener)
}

func (broker *Broker) CalculateNStates(args BrokerArgs, result *[][]byte) error {
	if args.Turns == 0 {
		*result = copySliceWorld(args.World)
		return nil
	}
	numNodes := 2
	nodes := make([]*rpc.Client, numNodes)
	port := 9876
	//Establish connection with node

	for i := 0; i < numNodes; i++ {
		node, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(port+i))
		if err != nil {
			fmt.Println("Error connecting:", err)
		}
		nodes[i] = node
		defer node.Close()
	}

	//Initialise variables
	world := args.World
	keypress := broker.KeyPresses
	pauseFlag := false

	//TODO: send work to the nodes
	for t := 0; t < args.Turns; t++ {
		//Update arguments
		//nodeArgs.World = world

		//Update struct data
		broker.mu.Lock()
		broker.turn = t
		broker.world = world
		broker.mu.Unlock()

		// TODO: handle key presses
		select {
		case key := <-keypress:
			switch key {
			case 'p':
				pauseFlag = !pauseFlag
			case 'k':
				fmt.Println("k - closing nodes and broker")
				var dummy bool
				for _, node := range nodes {
					_ = node.Call("Node.CloseNode", true, &dummy)
				}
				go func() {
					time.Sleep(100 * time.Millisecond)
					broker.Listener.Close()
				}() //allow time for the world to be sent back to the distributor
				*result = world
				return nil
			} //It still processes needlessly after q but trying to stop it causes race conditions
		default:
			if !pauseFlag {
				fmt.Println("Calculating Next State")
				chunks := chunkWorld(numNodes, world)
				calculatedChunks := make([][][]byte, numNodes)
				trimmedChunks := make([][][]byte, numNodes)

				wg := sync.WaitGroup{}
				wg.Add(numNodes)
				for i, node := range nodes {
					chunk := chunks[i]
					nodeArgs := NodeArgs{
						len(chunks[i]),
						len(chunks[i][0]),
						args.Turns,
						copySliceWorld(chunk)}
					go func(currentNode *rpc.Client, args NodeArgs, n int) {
						defer wg.Done()
						result := make([][]byte, args.ImageHeight)
						_ = currentNode.Call("Node.CalculateNextState", args, &result)
						//need to trim result
						calculatedChunks[n] = result
						//trim that bitch :D
						trimmedChunks[n] = result[1 : len(result)-1]
					}(node, nodeArgs, i)
				}
				wg.Wait()
				//assemble world
				var writeWorld [][]byte
				for _, chunk := range trimmedChunks {
					writeWorld = append(writeWorld, chunk...)
				}
				world = copySliceWorld(writeWorld)
			} else {
				fmt.Println("Paused")
				t--
			}

		}
	}

	*result = world
	return nil
}

func (broker *Broker) RegisterKey(key rune, result *bool) error {
	fmt.Println("recieved", key)
	broker.KeyPresses <- key
	return nil
}

func (broker *Broker) GetWorld(args bool, result *[][]byte) error {
	broker.mu.Lock()
	*result = broker.world
	broker.mu.Unlock()
	return nil
}

func (broker *Broker) GetAliveCells(args BrokerArgs, result *AliveCellsCount) error {
	//TODO: get the current number of alive cells
	//Get info from struct
	broker.mu.Lock()
	turns := broker.turn
	world := broker.world
	broker.mu.Unlock()

	//Calculate the number
	liveCells := calculateNumAliveCells(args.ImageHeight, args.ImageWidth, world)

	//Send the info back
	aliveCellsCount := AliveCellsCount{CompletedTurns: turns, CellsCount: liveCells}
	*result = aliveCellsCount
	return nil
}

func calculateNumAliveCells(imageHeight, imageWidth int, world [][]byte) int {
	numCells := 0
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			if world[y][x] == 255 {
				numCells++
			}
		}
	}
	return numCells
}

// Calculate chunks for the nodes
func chunkWorld(numNodes int, world [][]byte) [][][]byte {
	chunkResult := make([][][]byte, numNodes)
	initialHeight := len(world)
	width := len(world[0])
	chunkWeWant := initialHeight / numNodes
	chunkHalos := initialHeight % numNodes

	startY := 0
	for i := 0; i < numNodes; i++ {
		usedHeight := chunkWeWant
		if i < chunkHalos {
			usedHeight++
		}
		endY := startY + usedHeight

		chunk := make([][]byte, usedHeight+2)
		//startY := (i * chunkHeight) //- 1
		//if startY < 0 {
		//	startY = 0
		//}

		topHalo := (startY - 1 + initialHeight) % initialHeight
		bottomHalo := endY % initialHeight

		chunk[0] = make([]byte, width)
		copy(chunk[0], world[topHalo])

		for y := 0; y < usedHeight; y++ {
			chunk[y+1] = make([]byte, width)
			copy(chunk[y+1], world[startY+y])
		}

		chunk[usedHeight+1] = make([]byte, width)
		copy(chunk[usedHeight+1], world[bottomHalo])

		chunkResult[i] = chunk
		startY = endY
	}
	return chunkResult
}

func copySliceWorld(world [][]byte) [][]byte {
	copyWorld := make([][]byte, len(world))
	for i := range world {
		copyWorld[i] = make([]byte, len(world[i]))
		copy(copyWorld[i], world[i]) // copy row
	}
	return copyWorld
}
