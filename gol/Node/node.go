package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
)

type Node struct {
	listener net.Listener
}

type NodeArgs struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}

func main() {
	//TODO Listen for new connections
	port := flag.String("port", "9000", "Port to listen on")
	flag.Parse()

	node := new(Node)
	rpc.Register(node)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Println("Error listening:", err)
	}
	fmt.Println("Listening on port", *port, "...")

	node.listener = listener

	rpc.Accept(listener)
}

func (node *Node) CloseNode(args bool, result *bool) error {
	node.listener.Close()
	return nil
}

func (node *Node) CalculateNextState(args NodeArgs, result *[][]byte) error {
	fmt.Println("Calculating a turn, world size:", len(args.World), len(args.World[5]))
	world := args.World
	if args.Turns == 0 {
		*result = world
		return nil
	}

	copyWorld := copySliceWorld(world)
	for i := 1; i < len(world)-1; i++ { //each row
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
					newi := i + k
					newj := adjustLeftRight(j+m, args.ImageWidth)
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

	*result = copyWorld
	return nil
}

// wrapping is not handled here - handle it when sending the args
func (node *Node) CalculateNextState2(args NodeArgs, result *[][]byte) error {
	writeWorld := makeWorld(args.ImageHeight, args.ImageWidth)
	oldWorld := args.World
	//fmt.Println("Calculating state: old world", len(oldWorld), "x", len(oldWorld[0]), "| new world", len(writeWorld), "x", len(writeWorld[0]))
	for y := 0; y < len(oldWorld); y++ {
		for x := 0; x < len(oldWorld[0]); x++ {
			aliveNeighbours := numAliveNeighbours(y, x, oldWorld)
			switch aliveNeighbours {
			case 2: //cell with 2 neighbours is unaffected
				writeWorld[y][x] = oldWorld[y][x]
			case 3: //cell with 3 neighbours becomes alive
				writeWorld[y][x] = 255
			default: //otherwise it's dead
				writeWorld[y][x] = 0
			}
		}
	}
	*result = writeWorld
	return nil
}

func numAliveNeighbours(y, x int, world [][]byte) int {
	result := 0
	for yDif := -1; yDif <= 1; yDif++ {
		for xDif := -1; xDif <= 1; xDif++ {
			if xDif == 0 && yDif == 0 {
				continue
			}
			//vertical wrapping is handled by broker
			if y+yDif < 0 {
				continue
			}
			if y+yDif >= len(world) {
				continue
			}
			finalY := y + yDif

			//horizontal wrapping needs to be implemented
			finalX := x + xDif
			if finalX < 0 {
				finalX = len(world[0]) - 1
			}
			if finalX >= len(world[0]) {
				finalX = 0
			}

			if world[finalY][finalX] == 255 {
				result++
			}
		}
	}
	return result
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}

func adjustLeftRight(a, b int) int {
	return (a + b) % b
}

func adjustUpDown(a, b int) int {
	return (a + b) % b
}

func copySliceWorld(world [][]byte) [][]byte {
	copyWorld := make([][]byte, len(world))
	for i := range world {
		copyWorld[i] = make([]byte, len(world[i]))
		copy(copyWorld[i], world[i]) // copy row
	}
	return copyWorld
}
