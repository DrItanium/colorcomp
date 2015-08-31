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
	NumCpus   = 64
	Kilo      = 1024
	Meg       = Kilo * Kilo
	MemSize   = 64 * Meg
	µcoreSize = MemSize / 64
)

type µcore struct {
	index   int
	memory  Memory
	result  chan *unicornhat.Pixel
	done    chan int
	r, g, b byte
}

const (
	OpDelay = iota
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpSet
	OpRotate
	OpDrain
	OpFill
	OpDrainTo
	OpFillTo
)

func (this *µcore) Execute() {
	log.Printf("µcore %d: Walking through memory", this.index)
	saturationIncrease := func(val, compare byte) byte {
		if val < compare {
			return 255
		} else {
			return val
		}
	}
	saturationDecrease := func(val, compare byte) byte {
		if val > compare {
			return 0
		} else {
			return val
		}
	}
	for i := 0; i < len(this.memory); i += 4 {
		r, g, b := this.memory[i+red], this.memory[i+green], this.memory[i+blue]
		// check the control byte
		c := this.memory[i+control]
		switch c {
		case OpDelay:
			MillisecondDelay(time.Duration((r + g + b) % 128))
		case OpAdd:
			this.r = saturationIncrease(this.r+r, this.r)
			this.g = saturationIncrease(this.g+g, this.g)
			this.b = saturationIncrease(this.b+b, this.b)
		case OpSub:
			this.r = saturationDecrease(this.r-r, this.r)
			this.g = saturationDecrease(this.g-g, this.g)
			this.b = saturationDecrease(this.b-b, this.b)
		case OpMul:
			this.r *= r
			this.b *= b
			this.g *= g
		case OpSet:
			this.r = r
			this.g = g
			this.b = b
		case OpDiv:
			if r != 0 {
				this.r /= r
			}
			if g != 0 {
				this.g /= g
			}
			if b != 0 {
				this.b /= b
			}
		case OpRotate:
			cr, cg, cb := this.r, this.g, this.b
			this.r = cb
			this.g = cr
			this.b = cg
		case OpDrain:
			// drain the pixel out each generation
			for this.g > 0 || this.b > 0 || this.r > 0 {
				this.r = saturationDecrease(this.r-1, this.r)
				this.g = saturationDecrease(this.g-1, this.g)
				this.b = saturationDecrease(this.b-1, this.b)
				this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
			}
			pixel := unicornhat.NewPixel(this.r, this.g, this.b)
			this.result <- pixel
			this.result <- pixel
			// once finished set the pixel to the r,g,b values in the instruction
			this.r, this.g, this.b = r, g, b
			this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
		case OpDrainTo:
			// drain the pixel out each generation
			for this.g > g || this.b > b || this.r > r {
				if this.r > r {
					this.r = saturationDecrease(this.r-1, this.r)
				}
				if this.g > g {
					this.g = saturationDecrease(this.g-1, this.g)
				}
				if this.b > b {
					this.b = saturationDecrease(this.b-1, this.b)
				}
				this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
			}
			pixel := unicornhat.NewPixel(this.r, this.g, this.b)
			this.result <- pixel
			this.result <- pixel
			// once finished set the pixel to the r,g,b values in the instruction
			this.r, this.g, this.b = r, g, b
			this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
		case OpFill:
			for this.g < 255 || this.b < 255 || this.r < 255 {
				this.r = saturationIncrease(this.r+1, this.r)
				this.g = saturationIncrease(this.g+1, this.g)
				this.b = saturationIncrease(this.b+1, this.b)
				this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
			}
			pixel := unicornhat.NewPixel(this.r, this.g, this.b)
			this.result <- pixel
			this.result <- pixel
			// once finished set the pixel to the r,g,b values in the instruction
			this.r, this.g, this.b = r, g, b
			this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
		case OpFillTo:
			for this.g < g || this.b < b || this.r < r {
				if this.r < r {
					this.r = saturationIncrease(this.r+1, this.r)
				}
				if this.g < g {
					this.g = saturationIncrease(this.g+1, this.g)
				}
				if this.b < b {
					this.b = saturationIncrease(this.b+1, this.b)
				}
				this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
			}
			pixel := unicornhat.NewPixel(this.r, this.g, this.b)
			this.result <- pixel
			this.result <- pixel
			// once finished set the pixel to the r,g,b values in the instruction
			this.r, this.g, this.b = r, g, b
			this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
		default:
		}
		this.result <- unicornhat.NewPixel(this.r, this.g, this.b)
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

const (
	red = iota
	green
	blue
	control
)

func main() {
	var cpus []*µcore
	var memory Memory
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer unicornhat.Shutdown()
	unicornhat.SetBrightness(unicornhat.DefaultBrightness / 2)
	if err := unicornhat.Initialize(); err != nil {
		fmt.Println(err)
		return
	}
	unicornhat.ClearLEDBuffer()
	unicornhat.Show()
	cpus = make([]*µcore, NumCpus)
	memory = make(Memory, MemSize)
	log.Print("Randomizing memory")
	fn := func(v int) byte {
		return byte(v + rand.Int())
	}
	for i, j := 0, 0; i < len(memory); i, j = i+4, j+1 {
		memory[i+red] = fn(j)
		memory[i+green] = fn(j)
		memory[i+blue] = fn(j)
		memory[i+control] = fn(j)
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
	time.Sleep(msec * time.Millisecond)
}
