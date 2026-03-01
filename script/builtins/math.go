package builtins

import (
	"fmt"
	"math"
	"math/rand"

	tengo "github.com/d5/tengo/v2"
)

func absFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "abs",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("abs: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("abs: argument must be a number")
			}
			return numericResult(math.Abs(v), args[0]), nil
		},
	}
}

func minFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "min",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("min: expected 2 arguments")
			}
			a, ok1 := toFloat64(args[0])
			b, ok2 := toFloat64(args[1])
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("min: arguments must be numbers")
			}
			if a <= b {
				return numericResult(a, args[0]), nil
			}
			return numericResult(b, args[1]), nil
		},
	}
}

func maxFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "max",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("max: expected 2 arguments")
			}
			a, ok1 := toFloat64(args[0])
			b, ok2 := toFloat64(args[1])
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("max: arguments must be numbers")
			}
			if a >= b {
				return numericResult(a, args[0]), nil
			}
			return numericResult(b, args[1]), nil
		},
	}
}

func signFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "sign",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("sign: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("sign: argument must be a number")
			}
			switch {
			case v > 0:
				return &tengo.Int{Value: 1}, nil
			case v < 0:
				return &tengo.Int{Value: -1}, nil
			default:
				return &tengo.Int{Value: 0}, nil
			}
		},
	}
}

func roundFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "round",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("round: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("round: argument must be a number")
			}
			return &tengo.Int{Value: int64(math.Round(v))}, nil
		},
	}
}

func floorFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "floor",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("floor: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("floor: argument must be a number")
			}
			return &tengo.Int{Value: int64(math.Floor(v))}, nil
		},
	}
}

func ceilFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "ceil",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("ceil: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("ceil: argument must be a number")
			}
			return &tengo.Int{Value: int64(math.Ceil(v))}, nil
		},
	}
}

func clampFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "clamp",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 3 {
				return tengo.UndefinedValue, fmt.Errorf("clamp: expected 3 arguments (val, min, max)")
			}
			val, ok1 := toFloat64(args[0])
			lo, ok2 := toFloat64(args[1])
			hi, ok3 := toFloat64(args[2])
			if !ok1 || !ok2 || !ok3 {
				return tengo.UndefinedValue, fmt.Errorf("clamp: arguments must be numbers")
			}
			if val < lo {
				val = lo
			} else if val > hi {
				val = hi
			}
			return numericResult(val, args[0]), nil
		},
	}
}

func lerpFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "lerp",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 3 {
				return tengo.UndefinedValue, fmt.Errorf("lerp: expected 3 arguments (a, b, t)")
			}
			a, ok1 := toFloat64(args[0])
			b, ok2 := toFloat64(args[1])
			t, ok3 := toFloat64(args[2])
			if !ok1 || !ok2 || !ok3 {
				return tengo.UndefinedValue, fmt.Errorf("lerp: arguments must be numbers")
			}
			return &tengo.Float{Value: a + (b-a)*t}, nil
		},
	}
}

func mapRangeFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "map_range",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 5 {
				return tengo.UndefinedValue, fmt.Errorf("map_range: expected 5 arguments (val, in_min, in_max, out_min, out_max)")
			}
			nums := make([]float64, 5)
			for i, a := range args {
				f, ok := toFloat64(a)
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("map_range: all arguments must be numbers")
				}
				nums[i] = f
			}
			val, inMin, inMax, outMin, outMax := nums[0], nums[1], nums[2], nums[3], nums[4]
			if inMax == inMin {
				return tengo.UndefinedValue, fmt.Errorf("map_range: in_min and in_max must differ")
			}
			return &tengo.Float{Value: (val-inMin)/(inMax-inMin)*(outMax-outMin) + outMin}, nil
		},
	}
}

func sqrtFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "sqrt",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("sqrt: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("sqrt: argument must be a number")
			}
			return &tengo.Float{Value: math.Sqrt(v)}, nil
		},
	}
}

func powFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "pow",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("pow: expected 2 arguments (base, exp)")
			}
			base, ok1 := toFloat64(args[0])
			exp, ok2 := toFloat64(args[1])
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("pow: arguments must be numbers")
			}
			return &tengo.Float{Value: math.Pow(base, exp)}, nil
		},
	}
}

func sinFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "sin",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("sin: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("sin: argument must be a number")
			}
			return &tengo.Float{Value: math.Sin(v)}, nil
		},
	}
}

func cosFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "cos",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("cos: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("cos: argument must be a number")
			}
			return &tengo.Float{Value: math.Cos(v)}, nil
		},
	}
}

func tanFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "tan",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("tan: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("tan: argument must be a number")
			}
			return &tengo.Float{Value: math.Tan(v)}, nil
		},
	}
}

func atan2Func() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "atan2",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("atan2: expected 2 arguments (y, x)")
			}
			y, ok1 := toFloat64(args[0])
			x, ok2 := toFloat64(args[1])
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("atan2: arguments must be numbers")
			}
			return &tengo.Float{Value: math.Atan2(y, x)}, nil
		},
	}
}

func hypotFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "hypot",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("hypot: expected 2 arguments (a, b)")
			}
			a, ok1 := toFloat64(args[0])
			b, ok2 := toFloat64(args[1])
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("hypot: arguments must be numbers")
			}
			return &tengo.Float{Value: math.Hypot(a, b)}, nil
		},
	}
}

func isNaNFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_nan",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_nan: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.FalseValue, nil
			}
			if math.IsNaN(v) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func isInfFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "is_inf",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return tengo.UndefinedValue, fmt.Errorf("is_inf: expected 1 argument")
			}
			v, ok := toFloat64(args[0])
			if !ok {
				return tengo.FalseValue, nil
			}
			if math.IsInf(v, 0) {
				return tengo.TrueValue, nil
			}
			return tengo.FalseValue, nil
		},
	}
}

func randIntFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "rand_int",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("rand_int: expected 2 arguments (min, max)")
			}
			lo, ok1 := args[0].(*tengo.Int)
			hi, ok2 := args[1].(*tengo.Int)
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("rand_int: arguments must be integers")
			}
			if hi.Value < lo.Value {
				return tengo.UndefinedValue, fmt.Errorf("rand_int: max must be >= min")
			}
			return &tengo.Int{Value: lo.Value + rand.Int63n(hi.Value-lo.Value+1)}, nil
		},
	}
}

func randFloatFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "rand_float",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			return &tengo.Float{Value: rand.Float64()}, nil
		},
	}
}

func deadBandFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "dead_band",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return tengo.UndefinedValue, fmt.Errorf("dead_band: expected 2 arguments (val, threshold)")
			}
			val, ok1 := toFloat64(args[0])
			threshold, ok2 := toFloat64(args[1])
			if !ok1 || !ok2 {
				return tengo.UndefinedValue, fmt.Errorf("dead_band: arguments must be numbers")
			}
			if math.Abs(val) < threshold {
				return &tengo.Float{Value: 0}, nil
			}
			return &tengo.Float{Value: val}, nil
		},
	}
}

func haversineFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "haversine",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 4 {
				return tengo.UndefinedValue, fmt.Errorf("haversine: expected 4 arguments (lat1, lon1, lat2, lon2)")
			}
			lat1, ok1 := toFloat64(args[0])
			lon1, ok2 := toFloat64(args[1])
			lat2, ok3 := toFloat64(args[2])
			lon2, ok4 := toFloat64(args[3])
			if !ok1 || !ok2 || !ok3 || !ok4 {
				return tengo.UndefinedValue, fmt.Errorf("haversine: all arguments must be numbers")
			}
			const R = 6371000.0
			phi1 := lat1 * math.Pi / 180
			phi2 := lat2 * math.Pi / 180
			dPhi := (lat2 - lat1) * math.Pi / 180
			dLam := (lon2 - lon1) * math.Pi / 180
			a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
				math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLam/2)*math.Sin(dLam/2)
			c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
			return &tengo.Float{Value: R * c}, nil
		},
	}
}
