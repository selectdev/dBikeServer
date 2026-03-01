package builtins

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	tengo "github.com/d5/tengo/v2"

	"dbikeserver/db"
)

func StateFuncs(state map[string]tengo.Object, mu *sync.RWMutex, database *db.DB) []*tengo.UserFunction {
	return []*tengo.UserFunction{
		getStateFunc(state, mu),
		setStateFunc(state, mu, database),
		delStateFunc(state, mu, database),
		throttleFunc(state, mu),
		debounceFunc(state, mu),
		ewmaFunc(state, mu),
		pidUpdateFunc(state, mu),
	}
}

func getStateFunc(state map[string]tengo.Object, mu *sync.RWMutex) *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "get_state",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("get_state: expected 1 argument (key)")
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("get_state: key must be a string")
			}
			mu.RLock()
			v, exists := state[k.Value]
			mu.RUnlock()
			if !exists {
				return tengo.UndefinedValue, nil
			}
			return v, nil
		},
	}
}

func setStateFunc(state map[string]tengo.Object, mu *sync.RWMutex, database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "set_state",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("set_state: expected 2 arguments (key, val)")
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("set_state: key must be a string")
			}
			mu.Lock()
			state[k.Value] = args[1]
			mu.Unlock()

			if !strings.HasPrefix(k.Value, "__") {
				if data, err := json.Marshal(tengoObjToGo(args[1])); err == nil {
					database.Set("state:"+k.Value, data)
				}
			}

			return tengo.UndefinedValue, nil
		},
	}
}

func delStateFunc(state map[string]tengo.Object, mu *sync.RWMutex, database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "del_state",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("del_state: expected 1 argument (key)")
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("del_state: key must be a string")
			}
			mu.Lock()
			delete(state, k.Value)
			mu.Unlock()

			if !strings.HasPrefix(k.Value, "__") {
				database.Delete("state:" + k.Value)
			}

			return tengo.UndefinedValue, nil
		},
	}
}

func debounceFunc(state map[string]tengo.Object, mu *sync.RWMutex) *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "debounce",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("debounce: expected 2 arguments (key, delay_ms)")
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("debounce: key must be a string")
			}
			delay, ok := toFloat64(args[1])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("debounce: delay_ms must be a number")
			}
			stateKey := "__debounce." + k.Value
			now := time.Now().UnixMilli()

			mu.RLock()
			last, exists := state[stateKey]
			mu.RUnlock()

			mu.Lock()
			state[stateKey] = &tengo.Int{Value: now}
			mu.Unlock()

			if !exists {
				return tengo.FalseValue, nil
			}
			if lastInt, ok := last.(*tengo.Int); ok {
				if now-lastInt.Value >= int64(delay) {
					return tengo.TrueValue, nil
				}
			}
			return tengo.FalseValue, nil
		},
	}
}

func ewmaFunc(state map[string]tengo.Object, mu *sync.RWMutex) *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "ewma",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 3 {
				return tengo.UndefinedValue, fmt.Errorf("ewma: expected 3 arguments (key, val, alpha)")
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("ewma: key must be a string")
			}
			val, ok := toFloat64(args[1])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("ewma: val must be a number")
			}
			alpha, ok := toFloat64(args[2])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("ewma: alpha must be a number")
			}
			stateKey := "__ewma." + k.Value

			mu.RLock()
			prev, exists := state[stateKey]
			mu.RUnlock()

			var result float64
			if exists {
				if prevF, ok := toFloat64(prev); ok {
					result = alpha*val + (1-alpha)*prevF
				} else {
					result = val
				}
			} else {
				result = val
			}

			mu.Lock()
			state[stateKey] = &tengo.Float{Value: result}
			mu.Unlock()

			return &tengo.Float{Value: result}, nil
		},
	}
}

func pidUpdateFunc(state map[string]tengo.Object, mu *sync.RWMutex) *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "pid_update",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 6 {
				return tengo.UndefinedValue, fmt.Errorf("pid_update: expected 6 arguments (key, setpoint, measured, kp, ki, kd)")
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pid_update: key must be a string")
			}
			setpoint, ok := toFloat64(args[1])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pid_update: setpoint must be a number")
			}
			measured, ok := toFloat64(args[2])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pid_update: measured must be a number")
			}
			kp, ok := toFloat64(args[3])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pid_update: kp must be a number")
			}
			ki, ok := toFloat64(args[4])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pid_update: ki must be a number")
			}
			kd, ok := toFloat64(args[5])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pid_update: kd must be a number")
			}

			stateKey := "__pid." + k.Value

			mu.RLock()
			prev, exists := state[stateKey]
			mu.RUnlock()

			var integrator, prevError float64
			if exists {
				if m, ok := prev.(*tengo.Map); ok {
					if iv, ok := m.Value["i"].(*tengo.Float); ok {
						integrator = iv.Value
					}
					if pv, ok := m.Value["p"].(*tengo.Float); ok {
						prevError = pv.Value
					}
				}
			}

			err := setpoint - measured
			integrator += err
			derivative := err - prevError
			output := kp*err + ki*integrator + kd*derivative

			mu.Lock()
			state[stateKey] = &tengo.Map{Value: map[string]tengo.Object{
				"i": &tengo.Float{Value: integrator},
				"p": &tengo.Float{Value: err},
			}}
			mu.Unlock()

			return &tengo.Float{Value: output}, nil
		},
	}
}

func throttleFunc(state map[string]tengo.Object, mu *sync.RWMutex) *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "throttle",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("throttle: expected 2 arguments (key, delay_ms)")
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("throttle: key must be a string")
			}
			delay, ok := toFloat64(args[1])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("throttle: delay_ms must be a number")
			}
			stateKey := "__throttle." + k.Value
			now := time.Now().UnixMilli()

			mu.RLock()
			last, exists := state[stateKey]
			mu.RUnlock()

			if exists {
				if lastInt, ok := last.(*tengo.Int); ok {
					if now-lastInt.Value < int64(delay) {
						return tengo.FalseValue, nil
					}
				}
			}

			mu.Lock()
			state[stateKey] = &tengo.Int{Value: now}
			mu.Unlock()
			return tengo.TrueValue, nil
		},
	}
}
