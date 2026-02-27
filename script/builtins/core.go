package builtins

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tengo "github.com/d5/tengo/v2"

	"dbikeserver/util"
)

func logFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "log",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			parts := make([]string, len(args))
			for i, a := range args {
				parts[i] = a.String()
			}
			util.Logf("[script] %s", strings.Join(parts, " "))
			return tengo.UndefinedValue, nil
		},
	}
}

func nowMsFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "now_ms",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			return &tengo.Int{Value: time.Now().UnixMilli()}, nil
		},
	}
}

func timeSinceMsFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "time_since_ms",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("time_since_ms: expected 1 argument")
			}
			ts, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("time_since_ms: argument must be a number")
			}
			return &tengo.Int{Value: time.Now().UnixMilli() - int64(ts)}, nil
		},
	}
}

func sprintfFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "sprintf",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return &tengo.String{Value: ""}, nil
			}
			fmtStr, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("sprintf: first argument must be a string")
			}
			goArgs := make([]any, len(args)-1)
			for i, a := range args[1:] {
				goArgs[i] = tengoObjToGo(a)
			}
			return &tengo.String{Value: fmt.Sprintf(fmtStr.Value, goArgs...)}, nil
		},
	}
}

func jsonEncodeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "json_encode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("json_encode: expected 1 argument")
			}
			b, err := json.Marshal(tengoObjToGo(args[0]))
			if err != nil {
				return tengo.UndefinedValue, fmt.Errorf("json_encode: %w", err)
			}
			return &tengo.String{Value: string(b)}, nil
		},
	}
}

func jsonDecodeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "json_decode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("json_decode: expected 1 argument")
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("json_decode: argument must be a string")
			}
			var v any
			if err := json.Unmarshal([]byte(s.Value), &v); err != nil {
				return tengo.UndefinedValue, fmt.Errorf("json_decode: %w", err)
			}
			return goToTengo(v), nil
		},
	}
}
