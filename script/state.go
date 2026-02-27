package script

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tengo "github.com/d5/tengo/v2"

	"dbikeserver/script/builtins"
)

func (e *Engine) throttleFunc() *tengo.UserFunction {
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
			delay, ok := builtins.ToFloat64(args[1])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("throttle: delay_ms must be a number")
			}
			stateKey := "__throttle." + k.Value
			now := time.Now().UnixMilli()

			e.stateMu.RLock()
			last, exists := e.state[stateKey]
			e.stateMu.RUnlock()

			if exists {
				if lastInt, ok := last.(*tengo.Int); ok {
					if now-lastInt.Value < int64(delay) {
						return tengo.FalseValue, nil
					}
				}
			}

			e.stateMu.Lock()
			e.state[stateKey] = &tengo.Int{Value: now}
			e.stateMu.Unlock()
			return tengo.TrueValue, nil
		},
	}
}

func (e *Engine) getStateFunc() *tengo.UserFunction {
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
			e.stateMu.RLock()
			v, exists := e.state[k.Value]
			e.stateMu.RUnlock()
			if !exists {
				return tengo.UndefinedValue, nil
			}
			return v, nil
		},
	}
}

func (e *Engine) setStateFunc() *tengo.UserFunction {
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
			e.stateMu.Lock()
			e.state[k.Value] = args[1]
			e.stateMu.Unlock()

			if !strings.HasPrefix(k.Value, "__") {
				if data, err := json.Marshal(builtins.TengoObjToGo(args[1])); err == nil {
					e.db.Set("state:"+k.Value, data)
				}
			}

			return tengo.UndefinedValue, nil
		},
	}
}

func (e *Engine) delStateFunc() *tengo.UserFunction {
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
			e.stateMu.Lock()
			delete(e.state, k.Value)
			e.stateMu.Unlock()

			if !strings.HasPrefix(k.Value, "__") {
				e.db.Delete("state:" + k.Value)
			}

			return tengo.UndefinedValue, nil
		},
	}
}
