package main

import (
	"flag"
	"fmt"
	"github.com/DrItanium/unicornhat"
	"log"
	"math/rand"
	"runtime"
	"time"
)

var msecdelay = flag.Uint("msecdelay", 33, "Millisecond delay between unicornhat updates")
var debug = flag.Bool("debug", false, "Enable debug logging")
var xmas = flag.Bool("xmas", false, "Enable xmas mode")
var gscale = flag.Bool("greyscale", false, "Enable greyscale mode")
var drainfill = flag.Bool("drainfill", false, "Only use drain and fill operations!")
var purple = flag.Bool("purple", false, "Only use purple tones")
var blueOnly = flag.Bool("blueOnly", false, "Only use blue tones")

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
)

func saturationIncrease(val, compare byte) byte {
	if val < compare {
		return 255
	} else {
		return val
	}
}

func saturationDecrease(val, compare byte) byte {
	if val > compare {
		return 0
	} else {
		return val
	}
}
func (this *µcore) delay(r, g, b byte) {
	MillisecondDelay(time.Duration((r + g + b) % 128))
}
func (this *µcore) add(r, g, b byte, fn func(byte, byte) byte) {
	this.r = fn(this.r+r, this.r)
	this.g = fn(this.g+g, this.g)
	if !*xmas {
		this.b = fn(this.b+b, this.b)
	}
}

func (this *µcore) sub(r, g, b byte, fn func(byte, byte) byte) {
	this.r = fn(this.r-r, this.r)
	this.g = fn(this.g-g, this.g)
	if !*xmas {
		this.b = fn(this.b-b, this.b)
	}
}
func (this *µcore) mul(r, g, b byte) {
	this.r *= r
	this.g *= g
	if !*xmas {
		this.b *= b
	}
}
func (this *µcore) set(r, g, b byte) {
	this.r = r
	this.g = g
	if !*xmas {
		this.b = b
	}
}
func tryDiv(val *byte, divisor byte) {
	if divisor != 0 {
		*val /= divisor
	}
}
func (this *µcore) div(r, g, b byte) {
	tryDiv(&this.r, r)
	tryDiv(&this.g, g)
	if !*xmas {
		tryDiv(&this.b, b)
	}
}

func (this *µcore) Execute() {
	logPrint(fmt.Sprintf("µcore %d: Walking through memory", this.index))
	for i := 0; i < len(this.memory); i += 4 {
		r, g, b := this.memory[i+red], this.memory[i+green], this.memory[i+blue]
		// check the control byte
		c := this.memory[i+control]
		useRedOnly := *xmas && (r > g); // either red or green
		transformPixel := func(r, g, b byte) (byte, byte, byte) {
			var blue byte
			var green byte
			var red byte
			if *xmas {
				blue = 0
				green = 0
				red = 0
				if useRedOnly {
					red = r
				} else {
					green = g
				}
			} else {
				red = r
				green = g
				blue = b 
			}
			return red, green, blue
		}
		switch c {
		case OpDelay:
			this.delay(r, g, b)
		case OpAdd:
			this.add(r, g, b, saturationIncrease)
		case OpSub:
			this.sub(r, g, b, saturationDecrease)
		case OpMul:
			this.mul(r, g, b)
		case OpSet:
			this.set(r, g, b)
		case OpDiv:
			this.div(r, g, b)
		case OpRotate:
			if *xmas {
				this.set(this.g, this.r, this.b)
			} else {
				this.set(this.b, this.r, this.g)
			}
		case OpDrain:
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
				red, green, blue := transformPixel(this.r, this.g, this.b)
				this.result <- unicornhat.NewPixel(red, green, blue)
			}
			red, green, blue := transformPixel(this.r, this.g, this.b)
			pixel := unicornhat.NewPixel(red, green, blue)
			this.result <- pixel
			this.result <- pixel
		case OpFill:
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
				red, green, blue := transformPixel(this.r, this.g, this.b)
				this.result <- unicornhat.NewPixel(red, green, blue)
			}
			red, green, blue := transformPixel(this.r, this.g, this.b)
			pixel := unicornhat.NewPixel(red, green, blue)
			this.result <- pixel
			this.result <- pixel
		default:
		}
		red, green, blue := transformPixel(this.r, this.g, this.b)
		this.result <- unicornhat.NewPixel(red, green, blue)
	}
	logPrint(fmt.Sprintf("µcore %d: done walking", this.index))
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

func logPrint(msg string) {
	if *debug {
		log.Print(msg)
	}
}
func main() {
	flag.Parse()
	delay := time.Duration(*msecdelay)
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
	logPrint("Randomizing memory")
	for i, j := 0, 0; i < len(memory); i, j = i+4, j+1 {
		memory[i+red] = 0
		memory[i+blue] = 0
		memory[i+green] = 0
		if *xmas {
			offset := red
			if rand.Int()%2 == 0 {
				offset = red
			} else {
				offset = green
			}
			memory[i+offset] = byte(j+rand.Int())
		} else if *gscale {
			intensity := byte(j * rand.Int())
			memory[i+red] = intensity
			memory[i+green] = intensity
			memory[i+blue] = intensity
		} else if *purple {
			intensity := byte(j + rand.Int())
			memory[i+red] = intensity
			memory[i+blue] = intensity
		} else if *blueOnly {
			memory[i+blue] = byte(j + rand.Int())
		} else {
			memory[i+red] = byte(j + rand.Int())
			memory[i+green] = byte(j + rand.Int())
			memory[i+blue] = byte(j + rand.Int())
		}
		if *drainfill {
			if val := byte(j * rand.Int()); val%2 == 0 {
				memory[i+control] = OpFill
			} else {
				memory[i+control] = OpDrain
			}
		} else {
			memory[i+control] = byte(j + rand.Int())
		}
		//logPrint(fmt.Sprintf("@%d = {%d, %d, %d}", i, memory[i+red], memory[i+green], memory[i+blue]))
	}
	logPrint("Done randomizing memory")
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
			if c := cpus[i]; c == nil {
				continue
			} else {
				select {
				case value := <-c.result:
					unicornhat.SetPixelColor(c.index, value.R, value.G, value.B)
				case <-c.done:
					count++
					cpus[i] = nil
				default:
				}
			}
		}
		unicornhat.Show()
		MillisecondDelay(delay)
	}
}
func MillisecondDelay(msec time.Duration) {
	time.Sleep(msec * time.Millisecond)
}
