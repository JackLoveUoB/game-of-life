//Example useage:
//go test -run=^$ -bench=BenchmarkGol -timeout=100s -cpuprofile=cpu.prof 2>&1 | grep '^Benchmark' | tee results.out

package gol_test

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

const benchLength = 1000

func BenchmarkGol(b *testing.B) {

	log.SetOutput(io.Discard)

	os.Stdout = nil // Disable all program output apart from benchmark results
	p := gol.Params{
		Turns:       benchLength,
		Threads:     1,
		ImageWidth:  512,
		ImageHeight: 512,
	}
	name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			events := make(chan gol.Event)
			go gol.Run(p, events, nil)
			for range events {

			}
		}
	})
}
