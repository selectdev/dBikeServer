package builtins

import (
	"math"

	tengo "github.com/d5/tengo/v2"
)

func All() []*tengo.UserFunction {
	return []*tengo.UserFunction{
		logFunc(),
		nowMsFunc(),
		timeSinceMsFunc(),
		sprintfFunc(),
		jsonEncodeFunc(),
		jsonDecodeFunc(),
		absFunc(),
		minFunc(),
		maxFunc(),
		signFunc(),
		roundFunc(),
		floorFunc(),
		ceilFunc(),
		clampFunc(),
		lerpFunc(),
		mapRangeFunc(),
		sqrtFunc(),
		powFunc(),
		sinFunc(),
		cosFunc(),
		tanFunc(),
		atan2Func(),
		hypotFunc(),
		isNaNFunc(),
		isInfFunc(),
		randIntFunc(),
		randFloatFunc(),
		sumFunc(),
		avgFunc(),
		minOfFunc(),
		maxOfFunc(),
		sortArrayFunc(),
		uniqueFunc(),
		flattenFunc(),
		zipFunc(),
		sliceArrayFunc(),
		arrayContainsFunc(),
		reverseFunc(),
		keysFunc(),
		valuesFunc(),
		hasKeyFunc(),
		mergeFunc(),
		pickFunc(),
		omitFunc(),
		mapToPairsFunc(),
		pairsToMapFunc(),
		splitFunc(),
		joinFunc(),
		trimFunc(),
		toUpperFunc(),
		toLowerFunc(),
		containsFunc(),
		startsWithFunc(),
		endsWithFunc(),
		replaceFunc(),
		replaceAllFunc(),
		repeatFunc(),
		padLeftFunc(),
		padRightFunc(),
		isIntFunc(),
		isFloatFunc(),
		isStringFunc(),
		isBoolFunc(),
		isArrayFunc(),
		isMapFunc(),
		isBytesFunc(),
		isUndefinedFunc(),
		hexEncodeFunc(),
		hexDecodeFunc(),
		base64EncodeFunc(),
		base64DecodeFunc(),
		deadBandFunc(),
		haversineFunc(),
		formatDurationFunc(),
		openaiChatFunc(),
		openaiChatExFunc(),
	}
}

func Vars() map[string]any {
	return map[string]any{
		"PI": math.Pi,
		"E":  math.E,
	}
}


func ToFloat64(o tengo.Object) (float64, bool) {
	return toFloat64(o)
}


func TengoMapToGo(m *tengo.Map) map[string]any {
	return tengoMapToGo(m)
}


func TengoObjToGo(o tengo.Object) any {
	return tengoObjToGo(o)
}


func GoToTengo(v any) tengo.Object {
	return goToTengo(v)
}

func toFloat64(o tengo.Object) (float64, bool) {
	switch v := o.(type) {
	case *tengo.Float:
		return v.Value, true
	case *tengo.Int:
		return float64(v.Value), true
	}
	return 0, false
}

func numericResult(val float64, ref tengo.Object) tengo.Object {
	if _, ok := ref.(*tengo.Int); ok {
		return &tengo.Int{Value: int64(val)}
	}
	return &tengo.Float{Value: val}
}

func tengoObjToGo(o tengo.Object) any {
	switch v := o.(type) {
	case *tengo.String:
		return v.Value
	case *tengo.Int:
		return v.Value
	case *tengo.Float:
		return v.Value
	case *tengo.Bool:
		return !v.IsFalsy()
	case *tengo.Bytes:
		return v.Value
	case *tengo.Map:
		return tengoMapToGo(v)
	case *tengo.Array:
		arr := make([]any, len(v.Value))
		for i, elem := range v.Value {
			arr[i] = tengoObjToGo(elem)
		}
		return arr
	default:
		return nil
	}
}

func tengoMapToGo(m *tengo.Map) map[string]any {
	result := make(map[string]any, len(m.Value))
	for k, v := range m.Value {
		result[k] = tengoObjToGo(v)
	}
	return result
}

func goToTengo(v any) tengo.Object {
	switch val := v.(type) {
	case nil:
		return tengo.UndefinedValue
	case bool:
		if val {
			return tengo.TrueValue
		}
		return tengo.FalseValue
	case int:
		return &tengo.Int{Value: int64(val)}
	case int64:
		return &tengo.Int{Value: val}
	case float64:
		return &tengo.Float{Value: val}
	case string:
		return &tengo.String{Value: val}
	case []byte:
		return &tengo.Bytes{Value: val}
	case map[string]any:
		m := &tengo.Map{Value: make(map[string]tengo.Object, len(val))}
		for k, vv := range val {
			m.Value[k] = goToTengo(vv)
		}
		return m
	case []any:
		arr := &tengo.Array{Value: make([]tengo.Object, len(val))}
		for i, vv := range val {
			arr.Value[i] = goToTengo(vv)
		}
		return arr
	default:
		return tengo.UndefinedValue
	}
}
