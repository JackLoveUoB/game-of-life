package gol

//go:noinline
func findNeighbours(p Params, rowIndex, columnIndex int, chunkedWorld [][]byte) int {
	adjacentAlive := 0

	for yDif := -1; yDif <= 1; yDif++ {
		for xDif := -1; xDif <= 1; xDif++ {
			if yDif == 0 && xDif == 0 {
				continue
			}

			finalY := adjustUpDown(rowIndex+yDif, p.ImageHeight)

			finalX := adjustLeftRight(columnIndex+xDif, p.ImageWidth)

			if chunkedWorld[finalY][finalX] == 255 {
				adjacentAlive++
			}
		}
	}
	return adjacentAlive
}
