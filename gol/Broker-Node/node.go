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

	copyWorld := copy2D(world)
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
					newi := adjustUpDown(i+k, args.ImageHeight)
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

func adjustLeftRight(a, b int) int {
	return (a + b) % b
}

func adjustUpDown(a, b int) int {
	return (a + b) % b
}

func copy2D(world [][]byte) [][]byte {
	copyWorld := make([][]byte, len(world))
	for i := range world {
		copyWorld[i] = make([]byte, len(world[i]))
		copy(copyWorld[i], world[i]) // copy row
	}
	return copyWorld
}
