package gol

type workerTask struct {
	world  [][]byte
	startY int
	endY   int
	turn   int
	out    chan<- [][]byte
}

func workerNoShutDown(p Params, c distributorChannels, tasks <-chan workerTask) {
	for task := range tasks {
		chunkHeight := task.endY - task.startY

		chunky := makeWorld(chunkHeight+2, p.ImageWidth)

		for y := 0; y < chunkHeight+2; y++ {
			sauceY := task.startY + y - 1

			if sauceY < 0 {
				sauceY = sauceY + p.ImageHeight
			} else if sauceY >= p.ImageHeight {
				sauceY = sauceY - p.ImageHeight
			}
			copy(chunky[y], task.world[sauceY])
		}
		next := calculateNextState(c, p, p.Turns, task.turn, task.startY, chunky)
		trimItDown := next[1 : len(next)-1]

		task.out <- trimItDown
	}
}

//func worker(p Params, c distributorChannels, world [][]byte, startY, endY, turn int, out chan<- [][]byte) {
//	chunkHeight := endY - startY
//
//	chunk := makeWorld(chunkHeight+2, p.ImageWidth)
//
//	for y := 0; y < chunkHeight+2; y++ {
//		sauceY := startY + y - 1
//
//		if sauceY < 0 {
//			sauceY = sauceY + p.ImageHeight
//		} else if sauceY >= p.ImageHeight {
//			sauceY = sauceY - p.ImageHeight
//		}
//		copy(chunk[y], world[sauceY])
//	}
//	next := calculateNextState(c, p, p.Turns, turn, startY, chunk)
//	trimItDown := next[1 : len(next)-1]
//	out <- trimItDown
//}
