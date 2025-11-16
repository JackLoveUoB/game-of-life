package tests

import (
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

func BenchmarkGol_512x512x100_threads8(b *testing.B) {
	p := gol.Params{
		ImageWidth:  512,
		ImageHeight: 512,
		Turns:       100,
		Threads:     8,
	}
	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gol.Run(p, events, keyPresses)
	}
}
