// go test -run=^$ -bench=BenchmarkGol -benchtime=10s -cpuprofile=cpu.out

package main

import (
	"fmt"
	"io"
	"log"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

const benchLength2 = 1000

func BenchmarkGol2(b *testing.B) {
	// Silence logging from the program so it doesn't pollute the benchmark.
	log.SetOutput(io.Discard)

	for threads := 1; threads <= 12; threads++ {
		p := gol.Params{
			Turns:       benchLength2,
			Threads:     12,
			ImageWidth:  512,
			ImageHeight: 512,
		}

		name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// One simulation run per iteration.
				events := make(chan gol.Event)

				// Run the simulation in a goroutine.
				go gol.Run(p, events, nil) // nil keypress channel -> no keyboard events

				// Drain all events until Run closes the channel.
				for range events {
				}
			}
		})
	}
}
