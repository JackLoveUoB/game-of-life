package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type Broker struct {
	turn       int
	world      [][]byte
	mu         sync.Mutex
	KeyPresses chan rune
	Listener   net.Listener
	Ips        []string
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

type WorldState struct {
	Turn  int
	World [][]byte
}

type ipList []string

func (i *ipList) String() string {
	return fmt.Sprint(*i)
}

func (i *ipList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	//TODO: handle connections
	var ips ipList
	flag.Var(&ips, "ip", "IP address to include (can be repeated)")
	flag.Parse()

	broker := new(Broker)
	rpc.Register(broker)

	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		fmt.Println("Error listening:", err)
	}
	fmt.Println("Listening on port 1234...")

	broker.KeyPresses = make(chan rune, 10)
	broker.Listener = listener
	broker.Ips = ips

	rpc.Accept(listener)
}

func (broker *Broker) InitBroker(args BrokerArgs, result *bool) error {
	fmt.Println("Initialising Broker")
	broker.mu.Lock()
	broker.world = copySliceWorld(args.World)
	broker.turn = 0
	// drain any leftover keypresses so old keys don't affect this run
	func(ch chan rune) {
		for {
			select {
			case <-ch:
				//this gets rid of a press in ch
			default:
				return
			}
		}
	}(broker.KeyPresses)
	broker.mu.Unlock()
	fmt.Println("Broker initialised!")

	*result = true
	return nil
}

func (broker *Broker) CalculateNStates(args BrokerArgs, result *[][]byte) error {
	if args.Turns == 0 {
		fmt.Println("No turns, sending back initial world")
		*result = copySliceWorld(args.World)
		return nil
	}
	numNodes := len(broker.Ips)
	nodes := make([]*rpc.Client, numNodes)
	//Establish connection with node

	for i := 0; i < numNodes; i++ {
		node, err := rpc.Dial("tcp", broker.Ips[i])
		if err != nil {
			fmt.Println("Error connecting:", err)
		}
		nodes[i] = node
		defer node.Close()
	}

	//Initialise variables
	world := copySliceWorld(args.World)
	keypress := broker.KeyPresses
	pauseFlag := false

	//TODO: send work to the nodes
	fmt.Println("Calculating", args.Turns, "turns on world size", args.ImageHeight, "x", args.ImageWidth, "using", numNodes, "nodes...")
	for t := 0; t < args.Turns; t++ {
		//Update arguments
		//nodeArgs.World = world

		// TODO: handle key presses
		select {
		case key := <-keypress:
			switch key {
			case 'p':
				pauseFlag = !pauseFlag
				fmt.Println("Paused:", pauseFlag)
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
			case 'q':
				time.Sleep(100 * time.Millisecond) //to allow time for distributor to quit gracefully
				fmt.Println("Ending processing early!")
				*result = world
				return nil
			}
		default:
			if !pauseFlag {
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
				t--
			}
			//Update struct data
			broker.mu.Lock()
			broker.turn = t + 1
			broker.world = world
			broker.mu.Unlock()
		}
	}
	fmt.Println("Done! Sending world with size", len(world), "x", len(world[0]))
	*result = world
	return nil
}

func (broker *Broker) RegisterKey(key rune, result *bool) error {
	if key == 's' {
		// 's' is a distributor-only snapshot request; ignore it here.
		*result = true
		return nil
	}
	fmt.Println("recieved", string(key))
	broker.KeyPresses <- key
	return nil
}

func (broker *Broker) GetWorld(args bool, result *WorldState) error {
	broker.mu.Lock()
	world := copySliceWorld(broker.world)
	// log number of live cells at time of request
	live := calculateNumAliveCells(len(world), len((world)[0]), world)
	turn := broker.turn
	broker.mu.Unlock()
	fmt.Printf("GetWorld: sending world (alive=%d, turn=%d)\n", live, turn)
	*result = WorldState{World: world, Turn: turn}
	return nil
}

func (broker *Broker) GetAliveCells(args BrokerArgs, result *AliveCellsCount) error {
	//TODO: get the current number of alive cells
	//Get info from struct
	broker.mu.Lock()
	turns := broker.turn
	world := broker.world
	broker.mu.Unlock()

	//fmt.Println("Getting live cells for world:", len(world), "x", len(world[0]))

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
