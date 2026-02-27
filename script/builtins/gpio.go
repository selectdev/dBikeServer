package builtins

import (
	"fmt"

	tengo "github.com/d5/tengo/v2"

	"dbikeserver/gpio"
)

var errGPIOUnavailable = fmt.Errorf("gpio: not available on this platform")

func GPIOFuncs(g *gpio.GPIO) []*tengo.UserFunction {
	return []*tengo.UserFunction{
		gpioInputFunc(g),
		gpioOutputFunc(g),
		gpioHighFunc(g),
		gpioLowFunc(g),
		gpioToggleFunc(g),
		gpioReadFunc(g),
		gpioPullUpFunc(g),
		gpioPullDownFunc(g),
		gpioPullOffFunc(g),
		gpioPwmFunc(g),
		gpioPwmFreqFunc(g),
		gpioPwmDutyFunc(g),
		gpioDetectFunc(g),
		gpioEdgeFunc(g),
		gpioStopDetectFunc(g),
	}
}

func gpioPin(args []tengo.Object, name string) (uint8, error) {
	if len(args) < 1 {
		return 0, fmt.Errorf("%s: expected pin number", name)
	}
	n, ok := args[0].(*tengo.Int)
	if !ok {
		return 0, fmt.Errorf("%s: pin must be an int", name)
	}
	return uint8(n.Value), nil
}

func gpioInputFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_input", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_input")
		if err != nil {
			return nil, err
		}
		g.Input(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioOutputFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_output", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_output")
		if err != nil {
			return nil, err
		}
		g.Output(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioHighFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_high", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_high")
		if err != nil {
			return nil, err
		}
		g.High(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioLowFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_low", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_low")
		if err != nil {
			return nil, err
		}
		g.Low(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioToggleFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_toggle", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_toggle")
		if err != nil {
			return nil, err
		}
		g.Toggle(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioReadFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_read", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_read")
		if err != nil {
			return nil, err
		}
		return &tengo.Int{Value: int64(g.Read(pin))}, nil
	}}
}

func gpioPullUpFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_pull_up", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_pull_up")
		if err != nil {
			return nil, err
		}
		g.PullUp(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioPullDownFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_pull_down", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_pull_down")
		if err != nil {
			return nil, err
		}
		g.PullDown(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioPullOffFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_pull_off", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_pull_off")
		if err != nil {
			return nil, err
		}
		g.PullOff(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioPwmFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_pwm", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_pwm")
		if err != nil {
			return nil, err
		}
		g.Pwm(pin)
		return tengo.UndefinedValue, nil
	}}
}

func gpioPwmFreqFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_pwm_freq", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		if len(args) != 2 {
			return nil, fmt.Errorf("gpio_pwm_freq: want 2 args (pin, freq_hz)")
		}
		pin, err := gpioPin(args, "gpio_pwm_freq")
		if err != nil {
			return nil, err
		}
		freqObj, ok := args[1].(*tengo.Int)
		if !ok {
			return nil, fmt.Errorf("gpio_pwm_freq: freq must be int")
		}
		g.PwmFreq(pin, int(freqObj.Value))
		return tengo.UndefinedValue, nil
	}}
}

func gpioPwmDutyFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_pwm_duty", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		if len(args) != 3 {
			return nil, fmt.Errorf("gpio_pwm_duty: want 3 args (pin, duty, cycle)")
		}
		pin, err := gpioPin(args, "gpio_pwm_duty")
		if err != nil {
			return nil, err
		}
		dutyObj, ok := args[1].(*tengo.Int)
		if !ok {
			return nil, fmt.Errorf("gpio_pwm_duty: duty must be int")
		}
		cycleObj, ok := args[2].(*tengo.Int)
		if !ok {
			return nil, fmt.Errorf("gpio_pwm_duty: cycle must be int")
		}
		g.PwmDuty(pin, uint32(dutyObj.Value), uint32(cycleObj.Value))
		return tengo.UndefinedValue, nil
	}}
}

func gpioDetectFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_detect", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		if len(args) != 2 {
			return nil, fmt.Errorf("gpio_detect: want 2 args (pin, edge)")
		}
		pin, err := gpioPin(args, "gpio_detect")
		if err != nil {
			return nil, err
		}
		edgeObj, ok := args[1].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("gpio_detect: edge must be string (\"rise\", \"fall\", or \"any\")")
		}
		g.Detect(pin, edgeObj.Value)
		return tengo.UndefinedValue, nil
	}}
}

func gpioEdgeFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_edge", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_edge")
		if err != nil {
			return nil, err
		}
		if g.EdgeDetected(pin) {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}}
}

func gpioStopDetectFunc(g *gpio.GPIO) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "gpio_stop_detect", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if g == nil {
			return nil, errGPIOUnavailable
		}
		pin, err := gpioPin(args, "gpio_stop_detect")
		if err != nil {
			return nil, err
		}
		g.StopDetect(pin)
		return tengo.UndefinedValue, nil
	}}
}
