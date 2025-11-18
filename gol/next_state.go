package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

func calculateNextStateDirect(c distributorChannels, p Params, turns, turn, startY, endY int, world, next [][]byte) {
	if turns == 0 {
		return
	}
	listOfFlippedInner := []util.Cell{}

	for i := startY; i < endY; i++ { //INLINE FIND NEIGHBOURS FOR CPU TO TAKE OVER
		neighbourUpPointer := i - 1
		if neighbourUpPointer < 0 {
			neighbourUpPointer = p.ImageHeight - 1
		}
		neighbourDownPointer := i + 1
		if neighbourDownPointer >= p.ImageHeight {
			neighbourDownPointer = 0
		}
		neighbourRowUp := world[neighbourUpPointer]
		neighbourRow := world[i]
		neighbourRowDown := world[neighbourDownPointer]
		for j := 0; j < p.ImageWidth; j++ {
			neighbourLeftPointer := j - 1
			if neighbourLeftPointer < 0 {
				neighbourLeftPointer = p.ImageWidth - 1
			}
			neighbourRightPointer := j + 1
			if neighbourRightPointer >= p.ImageWidth {
				neighbourRightPointer = 0
			}
			oldCell := world[i][j]
			alive := false
			if oldCell == 255 {
				alive = true
			}
			//adjacentAlive := findNeighbours(p, i, j, world) //INLINE TIME
			adjacentAlive := 0
			if neighbourRowUp[neighbourLeftPointer] == 255 {
				adjacentAlive++
			}
			if neighbourRowUp[j] == 255 {
				adjacentAlive++
			}
			if neighbourRowUp[neighbourRightPointer] == 255 {
				adjacentAlive++
			}
			if neighbourRow[neighbourLeftPointer] == 255 {
				adjacentAlive++
			}
			if neighbourRow[neighbourRightPointer] == 255 {
				adjacentAlive++
			}
			if neighbourRowDown[neighbourLeftPointer] == 255 {
				adjacentAlive++
			}
			if neighbourRowDown[j] == 255 {
				adjacentAlive++
			}
			if neighbourRowDown[neighbourRightPointer] == 255 {
				adjacentAlive++
			}
			if alive == true {
				if adjacentAlive < 2 || adjacentAlive > 3 {
					next[i][j] = 0
					listOfFlippedInner = append(listOfFlippedInner, util.Cell{j, i})
				} else {
					next[i][j] = 255
				}
			} else {
				if adjacentAlive == 3 {
					next[i][j] = 255
					listOfFlippedInner = append(listOfFlippedInner, util.Cell{j, i})
				} else {
					next[i][j] = 0
				}
			}
		}
	}
	if len(listOfFlippedInner) > 0 {
		c.events <- CellsFlipped{
			turn,
			listOfFlippedInner,
		}
	}
}

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
	if len(listOfFlippedInner) > 0 {
		c.events <- CellsFlipped{
			turn,
			listOfFlippedInner,
		}
	}
	//c.events <- TurnComplete{
	//	turn,
	//}
	return copyWorld
}
