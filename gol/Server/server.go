package main

import(
    "net"
    "net/rpc"
    "fmt"
)

type Worker struct{}

type Args struct{
    ImageHeight, ImageWidth, Turns int
    World [][]byte
}

func main(){
    worker := new(Worker)
    rpc.Register(worker)

    listener, err := net.Listen("tcp", ":1234")
    if(err != nil) {fmt.Println("Error listening:", err)}
    defer listener.Close()

    fmt.Println("Listening on port 1234...")
    rpc.Accept(listener)

}

func (worker *Worker) CalculateNextState(args Args, result *[][]byte) error {
    fmt.Println("Calculating State")
	if args.Turns == 0 {
		*result = args.World
		return nil
	}
	copyWorld := deepCopy2D(args.World)
	for i := 0; i < len(args.World); i++ { //each row
		for j := 0; j < len(args.World[i]); j++ { //each cell
			alive := 0
			adjacentAlive := 0
			if args.World[i][j] == 255 {
				alive = 1
			}
			for k := -1; k <= 1; k++ {
				for m := -1; m <= 1; m++ {
					if k == 0 && m == 0 {
						continue
					}
					newi := adjustUpDown(i+k, args.ImageHeight)
					newj := adjustLeftRight(j+m, args.ImageWidth)
					if args.World[newi][newj] == 255 {
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
	*result = copyWorld
	return nil
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