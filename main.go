package main

import (
	"fmt"
	"github.com/DrItanium/unicornhat"
	"log"
	"math/rand"
	"runtime"
	"time"
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
	MemSize    = 16 * Meg
	µcoreSize  = MemSize / 64
)

type µcore struct {
	index  uint
	memory Memory
	done   chan int
}

func (this *µcore) Execute() {
	log.Printf("µcore %d: Randomizing memory", this.index)
	for i := 0; i < len(this.memory); i++ {
		this.memory[i] = byte(rand.Uint32())
	}
	log.Printf("µcore %d: Done randomizing memory", this.index)
	log.Printf("µcore %d: Walking through memory", this.index)
	for i := 0; i < len(this.memory); i += 4 {
		unicornhat.SetPixelColor(this.index, this.memory[0], this.memory[1], this.memory[2])
		MicrosecondDelay(time.Duration(this.memory[3]))
	}
	log.Printf("µcore %d: done walking", this.index)
	this.done <- 0
	close(this.done)
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
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer unicornhat.Terminate(0)
	unicornhat.SetBrightness(unicornhat.DefaultBrightness())
	unicornhat.Init(NumCpus)
	unicornhat.ClearLEDBuffer()
	cpus = make([]*µcore, NumCpus)
	memory = make(Memory, MemSize)
	upperHalf := memory
	for i := 0; i < 64; i++ {
		targetMem := upperHalf[:µcoreSize]
		cpus[i] = New(uint(i), targetMem)
		go cpus[i].Execute()
		upperHalf = upperHalf[µcoreSize:]
	}

	count := 0
	for {
		if count >= 64 {
			break
		} else {
			for i := 0; i < 64; i++ {
				c := cpus[i].done
				select {
				case <-c:
					fmt.Println("done")
					count++
				default:
					unicornhat.Show()
					MicrosecondDelay(10)
				}
			}
			unicornhat.Show()
			MicrosecondDelay(10)
		}
	}
}

func MicrosecondDelay(usec time.Duration) {
	time.Sleep(usec * time.Microsecond)
}
