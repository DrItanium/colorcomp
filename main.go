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
	MemSize    = 8 * Meg
	µcoreSize  = MemSize / 64
)

type µcore struct {
	index  int
	memory Memory
	result chan *unicornhat.Pixel
	done   chan int
}

func (this *µcore) Execute() {
	log.Printf("µcore %d: Walking through memory", this.index)
	for i := 0; i < len(this.memory); i += 4 {
		this.result <- unicornhat.NewPixel(this.memory[i+0], this.memory[i+1], this.memory[i+2])
		MillisecondDelay(this.memory[i+3] % 17)
	}
	log.Printf("µcore %d: done walking", this.index)
	this.done <- 0
	close(this.done)
}

func New(index int, memory Memory) *µcore {
	var c µcore
	c.memory = memory
	c.index = index
	c.done = make(chan int)
	c.result = make(chan *unicornhat.Pixel)
	return &c
}

func main() {
	var cpus []*µcore
	var memory Memory
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer unicornhat.Shutdown()
	unicornhat.SetBrightness(128)
	if err := unicornhat.Initialize(); err != nil {
		fmt.Println(err)
		return
	}
	unicornhat.ClearLEDBuffer()
	unicornhat.Show()
	cpus = make([]*µcore, NumCpus)
	memory = make(Memory, MemSize)
	log.Print("Randomizing memory")
	for i := 0; i < len(memory); i++ {
		memory[i] = byte(rand.Uint32()) % 64
	}
	log.Print("Done randomizing memory")
	upperHalf := memory
	for i := 0; i < 64; i++ {
		targetMem := upperHalf[:µcoreSize]
		cpus[i] = New(i, targetMem)
		go cpus[i].Execute()
		upperHalf = upperHalf[µcoreSize:]
	}

	count := 0
	for {
		if count >= 64 {
			break
		}
		for i := 0; i < 64; i++ {
			c := cpus[i]
			if c == nil {
				continue
			}
			select {
			case value := <-c.result:
				unicornhat.SetPixelColor(c.index, value.R, value.G, value.B)
			case <-c.done:
				fmt.Println("done")
				count++
				cpus[i] = nil
			default:
			}
		}
		unicornhat.Show()
		MillisecondDelay(33)
	}
}
func MillisecondDelay(msec time.Duration) {
	time.Sleep(usec * time.Millisecond)
}
