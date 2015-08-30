package main

import (
	"fmt"
	"github.com/DrItanium/unicornhat"
	"math"
)

type Memory []byte
type Word uint32

const (
	NumCpus    = 64
	Kilo       = 1024
	Meg        = Kilo * Kilo
	MemorySize = 128 * Meg
	CpuStart   = 64 * Meg
	OneMeg     = 1 * Meg
)

type µcore struct {
	index  uint
	memory Memory
	done   chan int
}

func (this *µcore) Execute() {
	for i := 0; i < this.memory; i += 4 {
		unicornhat.SetPixelColor(this.index, this.memory[0], this.memory[1], this.memory[2])
	}
	this.done <- 0
}

func New(index uint, memory Memory) *µcore {
	var c µcore
	c.memory = memory
	c.index = index
	c.done = make(chan int)
	return &c
}

func main() {
	var cpus []*µcore
	var memory Memory
	defer unicornhat.Terminate(0)
	unicornhat.SetBrightness(unicornhat.DefaultBrightness())
	unicornhat.Init(NumCpus)
	unicornhat.ClearLEDBuffer()
	cpus = make([]*µcore, NumCpus)
	memory = make(Memory, 128*(1024*1024))
	upperHalf := memory[CpuStart:]
	for i := 0; i < 64; i++ {
		targetMem := upperHalf[:OneMeg]
		cpus[i] = New(uint(i), targetMem)
		go cpus[i].Execute()
		upperHalf = upperHalf[OneMeg:]
	}

	for i := 0; i < len(memory); i++ {
		memory[i] = byte(math.Uint32())
	}
	count := 0
	for {
		if count == 64 {
			break
		} else {
			for i := 0; i < 64; i++ {
				c := cpus[i].done
				select {
				case <-c:
					count++
				default:
				}
			}
			unicornhat.Show()
			microsecond_delay(10)
		}
	}
}
