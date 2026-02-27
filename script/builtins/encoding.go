package builtins

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	tengo "github.com/d5/tengo/v2"
)

func hexEncodeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "hex_encode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("hex_encode: expected 1 argument")
			}
			var b []byte
			switch v := args[0].(type) {
			case *tengo.Bytes:
				b = v.Value
			case *tengo.String:
				b = []byte(v.Value)
			default:
				return tengo.UndefinedValue, fmt.Errorf("hex_encode: argument must be bytes or string")
			}
			return &tengo.String{Value: hex.EncodeToString(b)}, nil
		},
	}
}

func hexDecodeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "hex_decode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("hex_decode: expected 1 argument")
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("hex_decode: argument must be a string")
			}
			b, err := hex.DecodeString(s.Value)
			if err != nil {
				return tengo.UndefinedValue, fmt.Errorf("hex_decode: %w", err)
			}
			return &tengo.Bytes{Value: b}, nil
		},
	}
}

func base64EncodeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "base64_encode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("base64_encode: expected 1 argument")
			}
			var b []byte
			switch v := args[0].(type) {
			case *tengo.Bytes:
				b = v.Value
			case *tengo.String:
				b = []byte(v.Value)
			default:
				return tengo.UndefinedValue, fmt.Errorf("base64_encode: argument must be bytes or string")
			}
			return &tengo.String{Value: base64.StdEncoding.EncodeToString(b)}, nil
		},
	}
}

func base64DecodeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "base64_decode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("base64_decode: expected 1 argument")
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("base64_decode: argument must be a string")
			}
			b, err := base64.StdEncoding.DecodeString(s.Value)
			if err != nil {
				return tengo.UndefinedValue, fmt.Errorf("base64_decode: %w", err)
			}
			return &tengo.Bytes{Value: b}, nil
		},
	}
}
