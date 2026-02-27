package gpio

import rpio "github.com/stianeikeland/go-rpio/v4"

type GPIO struct{}

func Open() (*GPIO, error) {
	return &GPIO{}, rpio.Open()
}

func (g *GPIO) Close() error {
	return rpio.Close()
}

func (g *GPIO) Input(pin uint8)  { rpio.Pin(pin).Input() }
func (g *GPIO) Output(pin uint8) { rpio.Pin(pin).Output() }

func (g *GPIO) High(pin uint8)   { rpio.Pin(pin).High() }
func (g *GPIO) Low(pin uint8)    { rpio.Pin(pin).Low() }
func (g *GPIO) Toggle(pin uint8) { rpio.Pin(pin).Toggle() }
func (g *GPIO) Read(pin uint8) int {
	if rpio.Pin(pin).Read() == rpio.High {
		return 1
	}
	return 0
}

func (g *GPIO) PullUp(pin uint8)   { rpio.Pin(pin).PullUp() }
func (g *GPIO) PullDown(pin uint8) { rpio.Pin(pin).PullDown() }
func (g *GPIO) PullOff(pin uint8)  { rpio.Pin(pin).PullOff() }

func (g *GPIO) Pwm(pin uint8)                         { rpio.Pin(pin).Pwm() }
func (g *GPIO) PwmFreq(pin uint8, freq int)           { rpio.Pin(pin).Freq(freq) }
func (g *GPIO) PwmDuty(pin uint8, duty, cycle uint32) { rpio.Pin(pin).DutyCycle(duty, cycle) }

func (g *GPIO) Detect(pin uint8, edge string) {
	var e rpio.Edge
	switch edge {
	case "rise":
		e = rpio.RiseEdge
	case "fall":
		e = rpio.FallEdge
	default:
		e = rpio.AnyEdge
	}
	rpio.Pin(pin).Detect(e)
}

func (g *GPIO) EdgeDetected(pin uint8) bool { return rpio.Pin(pin).EdgeDetected() }
func (g *GPIO) StopDetect(pin uint8)        { rpio.Pin(pin).Detect(rpio.NoEdge) }
