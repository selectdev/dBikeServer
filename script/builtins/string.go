package builtins

import (
	"fmt"
	"strings"

	tengo "github.com/d5/tengo/v2"
)

func splitFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "split",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("split: expected 2 arguments (str, sep)")
			}
			s, ok1 := args[0].(*tengo.String)
			sep, ok2 := args[1].(*tengo.String)
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("split: both arguments must be strings")
			}
			parts := strings.Split(s.Value, sep.Value)
			arr := &tengo.Array{Value: make([]tengo.Object, len(parts))}
			for i, p := range parts {
				arr.Value[i] = &tengo.String{Value: p}
			}
			return arr, nil
		},
	}
}

func joinFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "join",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("join: expected 2 arguments (arr, sep)")
			}
			arr, ok := args[0].(*tengo.Array)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("join: first argument must be an array")
			}
			sep, ok := args[1].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("join: second argument must be a string")
			}
			parts := make([]string, len(arr.Value))
			for i, v := range arr.Value {
				if s, ok := v.(*tengo.String); ok {
					parts[i] = s.Value
				} else {
					parts[i] = v.String()
				}
			}
			return &tengo.String{Value: strings.Join(parts, sep.Value)}, nil
		},
	}
}

func trimFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "trim",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("trim: expected 1 argument")
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("trim: argument must be a string")
			}
			return &tengo.String{Value: strings.TrimSpace(s.Value)}, nil
		},
	}
}

func toUpperFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "to_upper",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("to_upper: expected 1 argument")
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("to_upper: argument must be a string")
			}
			return &tengo.String{Value: strings.ToUpper(s.Value)}, nil
		},
	}
}

func toLowerFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "to_lower",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("to_lower: expected 1 argument")
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("to_lower: argument must be a string")
			}
			return &tengo.String{Value: strings.ToLower(s.Value)}, nil
		},
	}
}

func containsFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "contains",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("contains: expected 2 arguments (str, sub)")
			}
			s, ok1 := args[0].(*tengo.String)
			sub, ok2 := args[1].(*tengo.String)
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("contains: both arguments must be strings")
			}
			if strings.Contains(s.Value, sub.Value) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func startsWithFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "starts_with",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("starts_with: expected 2 arguments (str, prefix)")
			}
			s, ok1 := args[0].(*tengo.String)
			p, ok2 := args[1].(*tengo.String)
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("starts_with: both arguments must be strings")
			}
			if strings.HasPrefix(s.Value, p.Value) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func endsWithFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "ends_with",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("ends_with: expected 2 arguments (str, suffix)")
			}
			s, ok1 := args[0].(*tengo.String)
			sfx, ok2 := args[1].(*tengo.String)
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("ends_with: both arguments must be strings")
			}
			if strings.HasSuffix(s.Value, sfx.Value) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func replaceFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "replace",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 3 {
				return tengo.UndefinedValue, fmt.Errorf("replace: expected 3 arguments (str, old, new)")
			}
			s, ok1 := args[0].(*tengo.String)
			old, ok2 := args[1].(*tengo.String)
			newStr, ok3 := args[2].(*tengo.String)
			if !ok1 || !ok2 || !ok3 {
				return tengo.UndefinedValue, fmt.Errorf("replace: all arguments must be strings")
			}
			return &tengo.String{Value: strings.Replace(s.Value, old.Value, newStr.Value, 1)}, nil
		},
	}
}

func replaceAllFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "replace_all",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 3 {
				return tengo.UndefinedValue, fmt.Errorf("replace_all: expected 3 arguments (str, old, new)")
			}
			s, ok1 := args[0].(*tengo.String)
			old, ok2 := args[1].(*tengo.String)
			newStr, ok3 := args[2].(*tengo.String)
			if !ok1 || !ok2 || !ok3 {
				return tengo.UndefinedValue, fmt.Errorf("replace_all: all arguments must be strings")
			}
			return &tengo.String{Value: strings.ReplaceAll(s.Value, old.Value, newStr.Value)}, nil
		},
	}
}

func repeatFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "repeat",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("repeat: expected 2 arguments (str, n)")
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("repeat: first argument must be a string")
			}
			n, ok := args[1].(*tengo.Int)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("repeat: second argument must be an integer")
			}
			if n.Value < 0 {
				return tengo.UndefinedValue, fmt.Errorf("repeat: n must be non-negative")
			}
			return &tengo.String{Value: strings.Repeat(s.Value, int(n.Value))}, nil
		},
	}
}

func padLeftFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "pad_left",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 3 {
				return tengo.UndefinedValue, fmt.Errorf("pad_left: expected 3 arguments (str, width, pad)")
			}
			s, ok1 := args[0].(*tengo.String)
			w, ok2 := args[1].(*tengo.Int)
			p, ok3 := args[2].(*tengo.String)
			if !ok1 || !ok2 || !ok3 {
				return tengo.UndefinedValue, fmt.Errorf("pad_left: wrong argument types")
			}
			if len(p.Value) == 0 {
				return tengo.UndefinedValue, fmt.Errorf("pad_left: pad must not be empty")
			}
			result := s.Value
			for len([]rune(result)) < int(w.Value) {
				result = p.Value + result
			}
			return &tengo.String{Value: result}, nil
		},
	}
}

func padRightFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "pad_right",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 3 {
				return tengo.UndefinedValue, fmt.Errorf("pad_right: expected 3 arguments (str, width, pad)")
			}
			s, ok1 := args[0].(*tengo.String)
			w, ok2 := args[1].(*tengo.Int)
			p, ok3 := args[2].(*tengo.String)
			if !ok1 || !ok2 || !ok3 {
				return tengo.UndefinedValue, fmt.Errorf("pad_right: wrong argument types")
			}
			if len(p.Value) == 0 {
				return tengo.UndefinedValue, fmt.Errorf("pad_right: pad must not be empty")
			}
			result := s.Value
			for len([]rune(result)) < int(w.Value) {
				result = result + p.Value
			}
			return &tengo.String{Value: result}, nil
		},
	}
}
