package gol

func worker(p Params, c distributorChannels, world [][]byte, startY, endY int, turn int, out chan<- [][]byte) {
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth
	chunkHeight := endY - startY

	chunk := makeWorld(chunkHeight+2, imageWidth)

	for y := 0; y < chunkHeight+2; y++ {
		sauceY := (startY + y - 1 + imageHeight) % imageHeight
		copy(chunk[y], world[sauceY])
	}
	next := calculateNextState(c, chunkHeight+2, imageWidth, turn, chunk)
	trimItDown := next[1 : len(next)-1]
	out <- trimItDown
}
