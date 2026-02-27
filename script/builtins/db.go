package builtins

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tengo "github.com/d5/tengo/v2"

	"dbikeserver/db"
)

func DBFuncs(database *db.DB) []*tengo.UserFunction {
	return []*tengo.UserFunction{
		dbGetFunc(database),
		dbSetFunc(database),
		dbDelFunc(database),
		dbKeysFunc(database),
		dbLogFunc(database),
		dbLogsFunc(database),
		configGetFunc(database),
		configSetFunc(database),
		configDelFunc(database),
	}
}

func dbGetFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "db_get", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("db_get: want 1 arg, got %d", len(args))
		}
		key, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("db_get: key must be string")
		}
		val, found, err := database.Get("kv:" + key.Value)
		if err != nil || !found {
			return tengo.UndefinedValue, nil
		}
		var v any
		if err := json.Unmarshal(val, &v); err != nil {
			return tengo.UndefinedValue, nil
		}
		return goToTengo(v), nil
	}}
}

func dbSetFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "db_set", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("db_set: want 2 args, got %d", len(args))
		}
		key, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("db_set: key must be string")
		}
		data, err := json.Marshal(tengoObjToGo(args[1]))
		if err != nil {
			return nil, fmt.Errorf("db_set: marshal: %w", err)
		}
		return tengo.UndefinedValue, database.Set("kv:"+key.Value, data)
	}}
}

func dbDelFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "db_del", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("db_del: want 1 arg, got %d", len(args))
		}
		key, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("db_del: key must be string")
		}
		return tengo.UndefinedValue, database.Delete("kv:" + key.Value)
	}}
}

func dbKeysFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "db_keys", Value: func(args ...tengo.Object) (tengo.Object, error) {
		prefix := "kv:"
		if len(args) == 1 {
			s, ok := args[0].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("db_keys: prefix must be string")
			}
			prefix = "kv:" + s.Value
		}
		keys, err := database.ScanKeys(prefix)
		if err != nil {
			return nil, err
		}
		arr := &tengo.Array{Value: make([]tengo.Object, len(keys))}
		for i, k := range keys {
			arr.Value[i] = &tengo.String{Value: strings.TrimPrefix(k, "kv:")}
		}
		return arr, nil
	}}
}

func dbLogFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "db_log", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("db_log: want 2 args (topic, data), got %d", len(args))
		}
		topicObj, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("db_log: topic must be string")
		}
		data, err := json.Marshal(tengoObjToGo(args[1]))
		if err != nil {
			return nil, fmt.Errorf("db_log: marshal: %w", err)
		}
		key := fmt.Sprintf("log:%s:%020d", topicObj.Value, time.Now().UnixNano())
		return tengo.UndefinedValue, database.Set(key, data)
	}}
}

func dbLogsFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "db_logs", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("db_logs: want at least 1 arg (topic)")
		}
		topicObj, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("db_logs: topic must be string")
		}
		limit := 100
		if len(args) >= 2 {
			n, ok := args[1].(*tengo.Int)
			if !ok {
				return nil, fmt.Errorf("db_logs: limit must be int")
			}
			limit = int(n.Value)
		}
		pairs, err := database.ScanReverse("log:"+topicObj.Value+":", limit)
		if err != nil {
			return nil, err
		}
		arr := &tengo.Array{Value: make([]tengo.Object, 0, len(pairs))}
		for _, pair := range pairs {
			var v any
			if err := json.Unmarshal(pair[1], &v); err != nil {
				continue
			}
			arr.Value = append(arr.Value, goToTengo(v))
		}
		return arr, nil
	}}
}

func configGetFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "config_get", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("config_get: want 1 arg, got %d", len(args))
		}
		key, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("config_get: key must be string")
		}
		val, found, err := database.Get("config:" + key.Value)
		if err != nil || !found {
			return tengo.UndefinedValue, nil
		}
		return &tengo.String{Value: string(val)}, nil
	}}
}

func configSetFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "config_set", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("config_set: want 2 args, got %d", len(args))
		}
		key, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("config_set: key must be string")
		}
		val, ok := args[1].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("config_set: value must be string")
		}
		return tengo.UndefinedValue, database.Set("config:"+key.Value, []byte(val.Value))
	}}
}

func configDelFunc(database *db.DB) *tengo.UserFunction {
	return &tengo.UserFunction{Name: "config_del", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("config_del: want 1 arg, got %d", len(args))
		}
		key, ok := args[0].(*tengo.String)
		if !ok {
			return nil, fmt.Errorf("config_del: key must be string")
		}
		return tengo.UndefinedValue, database.Delete("config:" + key.Value)
	}}
}
