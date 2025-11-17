package gol

func worker(p Params, c distributorChannels, world [][]byte, startY, endY, turn int, out chan<- [][]byte) {
	chunkHeight := endY - startY

	chunk := makeWorld(chunkHeight+2, p.ImageWidth)

	for y := 0; y < chunkHeight+2; y++ {
		sauceY := startY + y - 1

		if sauceY < 0 {
			sauceY = sauceY + p.ImageHeight
		} else if sauceY >= p.ImageHeight {
			sauceY = sauceY - p.ImageHeight
		}
		copy(chunk[y], world[sauceY])
	}
	next := calculateNextState(c, p, p.Turns, turn, startY, chunk)
	trimItDown := next[1 : len(next)-1]
	out <- trimItDown
}
