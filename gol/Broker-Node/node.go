package main

type Node struct {
}

type NodeArgs struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}

func main() {
	//TODO Listen for new connections
}

func (node *Node) CalculateNextState(args NodeArgs, result *[][]byte) error {
	world := args.World
	if args.Turns == 0 {
		*result = world
		return nil
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

func deepCopy2D(world [][]byte) [][]byte {
	copyWorld := make([][]byte, len(world))
	for i := range world {
		copyWorld[i] = make([]byte, len(world[i]))
		copy(copyWorld[i], world[i]) // copy row
	}
	return copyWorld
}
