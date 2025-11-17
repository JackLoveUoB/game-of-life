package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type Broker struct {
	turn       int
	world      [][]byte
	mu         sync.Mutex
	KeyPresses chan rune
	Listener   net.Listener
	Ips        []string
	Client     *rpc.Client
	sendWg     sync.WaitGroup
	stopChan   chan bool
}

type BrokerArgs struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
	DistPort                       string
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

type NodeReturn struct {
	World   [][]byte
	Flipped []util.Cell
}

type CellsFlipped struct { // implements Event
	CompletedTurns int
	Cells          []util.Cell
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

	listener, err := net.Listen("tcp", ":1235")
	if err != nil {
		fmt.Println("Error listening:", err)
	}
	fmt.Println("Listening on port 1235...")

	stopBroker := make(chan bool)

	for {
		conn, _ := listener.Accept()

		select {
		case <-stopBroker:
			fmt.Println("Returned")
			return
		default:

		}
		go func() {
			// Create an RPC server instance just for THIS connection
			server := rpc.NewServer()

			// Create a brand-new Broker for THIS connection
			broker := new(Broker)
			broker.stopChan = stopBroker
			broker.sendWg.Add(1)
			broker.KeyPresses = make(chan rune, 10)
			broker.Listener = listener
			broker.Ips = ips

			_ = server.Register(broker)

			server.ServeConn(conn)
			broker.sendWg.Done()
		}()
	}
}

func (broker *Broker) InitBroker(args BrokerArgs, result *bool) error {
	fmt.Println("Initialising Broker")
	broker.mu.Lock()
	broker.Client, _ = rpc.Dial("tcp", args.DistPort)
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

func (broker *Broker) DisconnectClient(args bool, result *bool) error {
	broker.Client.Close()
	fmt.Println("Stopped connection with client")
	time.Sleep(500 * time.Millisecond)
	//add in some fault tolerance here
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
	fmt.Println("nodes:", numNodes)
	for i := 0; i < numNodes; i++ {
		node, err := rpc.Dial("tcp", broker.Ips[i])
		if err != nil {
			fmt.Println("Error connecting:", err)
		} else {
			fmt.Println("Connected to node on ip", broker.Ips[i])
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
				fmt.Println("p - paused:", pauseFlag)
			case 'k':
				fmt.Println("k - closing nodes and broker")
				var dummy bool
				for _, node := range nodes {
					_ = node.Call("Node.CloseNode", true, &dummy)
				}
				go func() {
					time.Sleep(100 * time.Millisecond)
					close(broker.stopChan)
					broker.Listener.Close()
				}() //allow time for the world to be sent back to the distributor
				*result = world
				return nil
			case 'q':
				broker.sendWg.Wait() //to allow time for distributor to quit gracefully
				//time.Sleep(500 * time.Millisecond)
				fmt.Println("q - ending processing early!")
				*result = world
				return nil
			}
		default:
			if !pauseFlag {
				chunks := chunkWorld(numNodes, world)
				calculatedChunks := make([][][]byte, numNodes)
				trimmedChunks := make([][][]byte, numNodes)
				var resultChunk [][]byte

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
						_ = currentNode.Call("Node.CalculateNextState2", args, &resultChunk)
						//need to trim result
						calculatedChunks[n] = resultChunk
						//trim that bitch :D
						trimmedChunks[n] = resultChunk[1 : len(resultChunk)-1]
						//nodeReturn.Flipped = nodeReturn.Flipped[1 : len(nodeReturn.Flipped)-1]
					}(node, nodeArgs, i)
				}
				wg.Wait()
				//assemble world
				var writeWorld [][]byte
				for _, chunk := range trimmedChunks {
					writeWorld = append(writeWorld, chunk...)
				}
				var dummy bool
				_ = broker.Client.Call("Client.TurnComplete", struct {
					Turn               int
					OldWorld, NewWorld [][]byte
				}{t, world, writeWorld}, &dummy)

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
		fmt.Println("s - handled client side")
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
	live := calculateNumAliveCells(len(world), len((world)[0]), world)
	turn := broker.turn
	broker.mu.Unlock()
	*result = WorldState{World: world, Turn: turn}
	fmt.Printf("GetWorld: sending world (alive=%d, turn=%d)\n", live, turn)
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

	fmt.Println("Currently on turn", turns, "with", liveCells, "alive cells")

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
	result := make([][][]byte, numNodes)
	imageHeight := len(world)
	chunkHeight := imageHeight / numNodes

	for i := 0; i < numNodes; i++ {
		start := (i * chunkHeight) - 1
		if start < 0 {
			start = 0
		}
		end := ((i + 1) * chunkHeight) + 1
		if i == numNodes-1 {
			end = imageHeight
		}
		chunk := copySliceWorld(world[start:end])

		if i == 0 {
			chunk = append([][]byte{world[len(world)-1]}, chunk...)
		}
		if i == numNodes-1 {
			chunk = append(chunk, world[0])
		}
		result[i] = chunk
	}
	return result
}

func copySliceWorld(world [][]byte) [][]byte {
	copyWorld := make([][]byte, len(world))
	for i := range world {
		copyWorld[i] = make([]byte, len(world[i]))
		copy(copyWorld[i], world[i]) // copy row
	}
	return copyWorld
}
