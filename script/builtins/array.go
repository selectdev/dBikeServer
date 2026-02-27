package builtins

import (
	"fmt"
	"sort"

	tengo "github.com/d5/tengo/v2"
)

func sumFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "sum",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("sum: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("sum: argument must be an array")
			}
			allInt := true
			var total float64
			for _, elem := range arr.Value {
				v, ok := toFloat64(elem)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("sum: all elements must be numbers")
				}
				if _, isInt := elem.(*tengo.Int); !isInt {
					allInt = false
				}
				total += v
			}
			if allInt {
				return &tengo.Int{Value: int64(total)}, nil
			}
			return &tengo.Float{Value: total}, nil
		},
	}
}

func avgFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "avg",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("avg: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("avg: argument must be an array")
			}
			if len(arr.Value) == 0 {
				return tengo.UndefinedValue, fmt.Errorf("avg: array is empty")
			}
			var total float64
			for _, elem := range arr.Value {
				v, ok := toFloat64(elem)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("avg: all elements must be numbers")
				}
				total += v
			}
			return &tengo.Float{Value: total / float64(len(arr.Value))}, nil
		},
	}
}

func minOfFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "min_of",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("min_of: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("min_of: argument must be an array")
			}
			if len(arr.Value) == 0 {
				return tengo.UndefinedValue, fmt.Errorf("min_of: array is empty")
			}
			best, ok := toFloat64(arr.Value[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("min_of: all elements must be numbers")
			}
			bestObj := arr.Value[0]
			for _, elem := range arr.Value[1:] {
				v, ok := toFloat64(elem)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("min_of: all elements must be numbers")
				}
				if v < best {
					best = v
					bestObj = elem
				}
			}
			return bestObj, nil
		},
	}
}

func maxOfFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "max_of",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("max_of: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("max_of: argument must be an array")
			}
			if len(arr.Value) == 0 {
				return tengo.UndefinedValue, fmt.Errorf("max_of: array is empty")
			}
			best, ok := toFloat64(arr.Value[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("max_of: all elements must be numbers")
			}
			bestObj := arr.Value[0]
			for _, elem := range arr.Value[1:] {
				v, ok := toFloat64(elem)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("max_of: all elements must be numbers")
				}
				if v > best {
					best = v
					bestObj = elem
				}
			}
			return bestObj, nil
		},
	}
}

func sortArrayFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "sort_array",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("sort_array: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("sort_array: argument must be an array")
			}
			out := make([]tengo.Object, len(arr.Value))
			copy(out, arr.Value)
			allNumeric := true
			for _, elem := range out {
				if _, ok := toFloat64(elem); !ok {
					allNumeric = false
					break
				}
			}
			if allNumeric {
				sort.Slice(out, func(i, j int) bool {
					a, _ := toFloat64(out[i])
					b, _ := toFloat64(out[j])
					return a < b
				})
			} else {
				sort.Slice(out, func(i, j int) bool {
					return out[i].String() < out[j].String()
				})
			}
			return &tengo.Array{Value: out}, nil
		},
	}
}

func uniqueFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "unique",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("unique: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("unique: argument must be an array")
			}
			seen := make(map[string]bool, len(arr.Value))
			out := make([]tengo.Object, 0, len(arr.Value))
			for _, elem := range arr.Value {
				key := elem.String()
				if !seen[key] {
					seen[key] = true
					out = append(out, elem)
				}
			}
			return &tengo.Array{Value: out}, nil
		},
	}
}

func flattenFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "flatten",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("flatten: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("flatten: argument must be an array")
			}
			out := make([]tengo.Object, 0, len(arr.Value))
			for _, elem := range arr.Value {
				if inner, ok := elem.(*tengo.Array); ok {
					out = append(out, inner.Value...)
				} else {
					out = append(out, elem)
				}
			}
			return &tengo.Array{Value: out}, nil
		},
	}
}

func zipFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "zip",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("zip: expected 2 arguments")
			}
			a, ok1 := args[0].(*tengo.Array)
			b, ok2 := args[1].(*tengo.Array)
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("zip: both arguments must be arrays")
			}
			n := len(a.Value)
			if len(b.Value) < n {
				n = len(b.Value)
			}
			out := &tengo.Array{Value: make([]tengo.Object, n)}
			for i := 0; i < n; i++ {
				out.Value[i] = &tengo.Array{Value: []tengo.Object{a.Value[i], b.Value[i]}}
			}
			return out, nil
		},
	}
}

func sliceArrayFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "slice_array",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 || len(args) > 3 {
				return tengo.UndefinedValue, fmt.Errorf("slice_array: expected 2 or 3 arguments (arr, start[, end])")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("slice_array: first argument must be an array")
			}
			startObj, ok := args[1].(*tengo.Int)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("slice_array: start must be an integer")
			}
			n := len(arr.Value)
			start := int(startObj.Value)
			end := n
			if len(args) == 3 {
				endObj, ok := args[2].(*tengo.Int)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("slice_array: end must be an integer")
				}
				end = int(endObj.Value)
			}
			if start < 0 {
				start = 0
			}
			if end > n {
				end = n
			}
			if start > end {
				start = end
			}
			out := make([]tengo.Object, end-start)
			copy(out, arr.Value[start:end])
			return &tengo.Array{Value: out}, nil
		},
	}
}

func arrayContainsFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "array_contains",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("array_contains: expected 2 arguments (array, val)")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("array_contains: first argument must be an array")
			}
			needle := args[1].String()
			for _, elem := range arr.Value {
				if elem.String() == needle {
					return tengo.TrueValue, nil
				}
			}
			return tengo.FalseValue, nil
		},
	}
}

func reverseFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "reverse",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("reverse: expected 1 argument")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("reverse: argument must be an array")
			}
			n := len(arr.Value)
			out := &tengo.Array{Value: make([]tengo.Object, n)}
			for i, v := range arr.Value {
				out.Value[n-1-i] = v
			}
			return out, nil
		},
	}
}
