package gol

import "uk.ac.bris.cs/gameoflife/util"

func calculateNextState(c distributorChannels, imageHeight, imageWidth, turns int, turn int, startY int, world [][]byte) [][]byte {
	if turns == 0 {
		return world
	}
	listOfAliveInner := []util.Cell{}
	copyWorld := deepCopy2D(world)
	for i := 0; i < len(world); i++ { //each row
		for j := 0; j < len(world[i]); j++ { //each cell
			alive := false
			adjacentAlive := 0
			if world[i][j] == 255 {
				alive = true
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
			if alive == true {
				if adjacentAlive < 2 || adjacentAlive > 3 {
					copyWorld[i][j] = 0
					listOfAliveInner = append(listOfAliveInner, newCell(j, i+startY))
				} else {
					copyWorld[i][j] = 255
				}
			} else {
				if adjacentAlive == 3 {
					copyWorld[i][j] = 255
					listOfAliveInner = append(listOfAliveInner, newCell(j, i+startY))
				}
			}

		}

	}
	//world = copyWorld
	c.events <- CellsFlipped{
		turn,
		listOfAliveInner,
	}
	//c.events <- TurnComplete{
	//	turn,
	//}
	return copyWorld
}
