package main

import (
	"fmt"
	"net/rpc"
)

type Broker struct {
}

type BrokerArgs struct {
	ImageHeight, ImageWidth, Turns int
}

type AliveCellsCount struct {
	CompletedTurns int
	CellsCount     int
}

func main() {
	//TO DO: handle connections
}

func (broker *Broker) CalculateNStates(args BrokerArgs, result *[][]byte) error {
	//Establish connection with node
	node, err := rpc.Dial("tcp", "localhost:9876")
	if err != nil {
		fmt.Println("Error connecting:", err)
	}
	defer node.Close()

	//Initialise variables
	world := makeWorld(args.ImageHeight, args.ImageWidth)
	nodeArgs := NodeArgs{}

	//TODO: send work to the node
	for t := 0; t < args.Turns; t++ {
		nodeArgs.World = world
		_ = node.Call("Node.CalculateNextState", nodeArgs, &world)
	}

	// TODO: handle key presses

	return nil
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}

func (broker *Broker) GetAliveCells(args BrokerArgs, result *AliveCellsCount) error {
	//TODO: get the current number of alive cells

	return nil
}
