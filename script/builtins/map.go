package builtins

import (
	"fmt"

	tengo "github.com/d5/tengo/v2"
)

func keysFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "keys",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("keys: expected 1 argument")
			}
			m, ok := args[0].(*tengo.Map)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("keys: argument must be a map")
			}
			arr := &tengo.Array{Value: make([]tengo.Object, 0, len(m.Value))}
			for k := range m.Value {
				arr.Value = append(arr.Value, &tengo.String{Value: k})
			}
			return arr, nil
		},
	}
}

func valuesFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "values",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("values: expected 1 argument")
			}
			m, ok := args[0].(*tengo.Map)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("values: argument must be a map")
			}
			arr := &tengo.Array{Value: make([]tengo.Object, 0, len(m.Value))}
			for _, v := range m.Value {
				arr.Value = append(arr.Value, v)
			}
			return arr, nil
		},
	}
}

func hasKeyFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "has_key",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("has_key: expected 2 arguments (map, key)")
			}
			m, ok := args[0].(*tengo.Map)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("has_key: first argument must be a map")
			}
			k, ok := args[1].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("has_key: key must be a string")
			}
			if _, exists := m.Value[k.Value]; exists {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func mergeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "merge",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("merge: expected 2 arguments")
			}
			m1, ok1 := args[0].(*tengo.Map)
			m2, ok2 := args[1].(*tengo.Map)
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("merge: both arguments must be maps")
			}
			out := &tengo.Map{Value: make(map[string]tengo.Object, len(m1.Value)+len(m2.Value))}
			for k, v := range m1.Value {
				out.Value[k] = v
			}
			for k, v := range m2.Value {
				out.Value[k] = v
			}
			return out, nil
		},
	}
}

func pickFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "pick",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return tengo.UndefinedValue, fmt.Errorf("pick: expected at least 2 arguments (map, key, ...)")
			}
			m, ok := args[0].(*tengo.Map)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pick: first argument must be a map")
			}
			out := &tengo.Map{Value: make(map[string]tengo.Object)}
			for _, kObj := range args[1:] {
				k, ok := kObj.(*tengo.String)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("pick: keys must be strings")
				}
				if v, exists := m.Value[k.Value]; exists {
					out.Value[k.Value] = v
				}
			}
			return out, nil
		},
	}
}

func omitFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "omit",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return tengo.UndefinedValue, fmt.Errorf("omit: expected at least 2 arguments (map, key, ...)")
			}
			m, ok := args[0].(*tengo.Map)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("omit: first argument must be a map")
			}
			drop := make(map[string]bool, len(args)-1)
			for _, kObj := range args[1:] {
				k, ok := kObj.(*tengo.String)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("omit: keys must be strings")
				}
				drop[k.Value] = true
			}
			out := &tengo.Map{Value: make(map[string]tengo.Object, len(m.Value))}
			for k, v := range m.Value {
				if !drop[k] {
					out.Value[k] = v
				}
			}
			return out, nil
		},
	}
}

func mapToPairsFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "map_to_pairs",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("map_to_pairs: expected 1 argument")
			}
			m, ok := args[0].(*tengo.Map)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("map_to_pairs: argument must be a map")
			}
			out := &tengo.Array{Value: make([]tengo.Object, 0, len(m.Value))}
			for k, v := range m.Value {
				pair := &tengo.Array{Value: []tengo.Object{&tengo.String{Value: k}, v}}
				out.Value = append(out.Value, pair)
			}
			return out, nil
		},
	}
}

func pairsToMapFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "pairs_to_map",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("pairs_to_map: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("pairs_to_map: argument must be an array")
			}
			out := &tengo.Map{Value: make(map[string]tengo.Object, len(arr.Value))}
			for i, elem := range arr.Value {
				pair, ok := elem.(*tengo.Array)
				if !ok || len(pair.Value) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("pairs_to_map: element %d is not a [key,val] pair", i)
				}
				k, ok := pair.Value[0].(*tengo.String)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("pairs_to_map: key at element %d must be a string", i)
				}
				out.Value[k.Value] = pair.Value[1]
			}
			return out, nil
		},
	}
}
