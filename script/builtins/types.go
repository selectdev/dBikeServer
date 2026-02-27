package builtins

import (
	"fmt"

	tengo "github.com/d5/tengo/v2"
)

func isIntFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_int",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_int: expected 1 argument")
			}
			if _, ok := args[0].(*tengo.Int); ok {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isFloatFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_float",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_float: expected 1 argument")
			}
			if _, ok := args[0].(*tengo.Float); ok {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isStringFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_string",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_string: expected 1 argument")
			}
			if _, ok := args[0].(*tengo.String); ok {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isBoolFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_bool",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_bool: expected 1 argument")
			}
			if _, ok := args[0].(*tengo.Bool); ok {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isArrayFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_array",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_array: expected 1 argument")
			}
			if _, ok := args[0].(*tengo.Array); ok {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isMapFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_map",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_map: expected 1 argument")
			}
			if _, ok := args[0].(*tengo.Map); ok {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isBytesFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_bytes",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_bytes: expected 1 argument")
			}
			if _, ok := args[0].(*tengo.Bytes); ok {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isUndefinedFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_undefined",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_undefined: expected 1 argument")
			}
			if args[0] == tengo.UndefinedValue {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}
