package main

import(
    "net"
    "net/rpc"
    "fmt"
    "sync"
)

type Server struct{
    turn int
    world [][]byte
    mu sync.Mutex
}

type Args struct{
    ImageHeight, ImageWidth, Turns int
    World [][]byte
}

type AliveCellsCount struct {
	CompletedTurns int
	CellsCount     int
}

func main(){
    server := new(Server)
    rpc.Register(server)

    listener, err := net.Listen("tcp", ":1234")
    if(err != nil) {fmt.Println("Error listening:", err)}
    defer listener.Close()

    fmt.Println("Listening on port 1234...")
    rpc.Accept(listener)

}

func (server *Server) GetAliveCells(args Args, result *AliveCellsCount) error{
    server.mu.Lock()
    turns := server.turn
    aliveCells := calculateNumAliveCells(args.ImageHeight, args.ImageWidth, server.world)
    server.mu.Unlock()
    *result = AliveCellsCount{turns, aliveCells}
    return nil
}

func (server *Server) CalculateNStates(args Args, result *[][]byte) error{
    fmt.Println("Calculating", args.Turns, "turns")

    //Calculate the states
    world := args.World
    for t := 0; t < args.Turns; t++{
        server.mu.Lock()
        server.turn = t
        server.world = world
        server.mu.Unlock()
        world = CalculateNextState(args.ImageHeight, args.ImageWidth, args.Turns, world)
    }
    *result = world
    return nil
}

func CalculateNextState(imageHeight, imageWidth, turns int, world [][]byte) [][]byte {
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

func calculateNumAliveCells(imageHeight, imageWidth int, world [][]byte) int {
	numCells := 0
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			if world[y][x] == 255 {
				numCells ++
			}
		}
	}
	return numCells
}