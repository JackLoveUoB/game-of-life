package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

//go:noinline
func calculateNextState(c distributorChannels, p Params, turns, turn, startY int, chunkedWorld [][]byte) [][]byte {
	if turns == 0 {
		return chunkedWorld
	}
	listOfFlippedInner := []util.Cell{}
	copyWorld := copySliceWorld(chunkedWorld)

	start := 0
	end := 0

	if len(chunkedWorld) == p.ImageHeight {
		start = 0
		end = len(chunkedWorld)
	} else {
		start = 1
		end = len(chunkedWorld) - 1
	}

	for i := start; i < end; i++ { //each row
		globalY := 0

		if len(chunkedWorld) == p.ImageHeight {
			globalY = i
		} else {
			globalY = i + startY - 1
		}

		for j := 0; j < len(chunkedWorld[i]); j++ { //each cell
			alive := false
			//adjacentAlive := 0

			if chunkedWorld[i][j] == 255 {
				alive = true
			}

			adjacentAlive := findNeighbours(p, i, j, chunkedWorld)

			if alive == true {
				if adjacentAlive < 2 || adjacentAlive > 3 {
					copyWorld[i][j] = 0
					listOfFlippedInner = append(listOfFlippedInner, util.Cell{j, globalY})
					//listOfFlippedInner = append(listOfFlippedInner, newCell(j, globalY))
				} else {
					copyWorld[i][j] = 255
				}
			} else {
				if adjacentAlive == 3 {
					listOfFlippedInner = append(listOfFlippedInner, util.Cell{j, globalY})
					copyWorld[i][j] = 255
					//listOfFlippedInner = append(listOfFlippedInner, newCell(j, globalY))
				}
			}
			if copyWorld[i][j] != chunkedWorld[i][j] {
				//fmt.Println(j, i+startY)
				//listOfFlippedInner = append(listOfFlippedInner, util.Cell{j, globalY})
			}

		}

	}
	//world = copyWorld
	c.events <- CellsFlipped{
		turn,
		listOfFlippedInner,
	}
	//c.events <- TurnComplete{
	//	turn,
	//}
	return copyWorld
}
