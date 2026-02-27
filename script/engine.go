package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tengo "github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"

	"dbikeserver/ble"
	"dbikeserver/db"
	"dbikeserver/gpio"
	"dbikeserver/script/builtins"
	"dbikeserver/util"
)

type Engine struct {
	nc      *ble.NotifyCharacteristic
	db      *db.DB
	gpio    *gpio.GPIO
	scripts map[string]*tengo.Compiled
	stateMu sync.RWMutex
	state   map[string]tengo.Object
}

func NewEngine(nc *ble.NotifyCharacteristic, database *db.DB, gp *gpio.GPIO, dir string) (*Engine, error) {
	e := &Engine{
		nc:      nc,
		db:      database,
		gpio:    gp,
		scripts: make(map[string]*tengo.Compiled),
		state:   make(map[string]tengo.Object),
	}

	pairs, err := database.Scan("state:")
	if err == nil {
		for _, pair := range pairs {
			key := strings.TrimPrefix(string(pair[0]), "state:")
			var v any
			if json.Unmarshal(pair[1], &v) == nil {
				e.state[key] = builtins.GoToTengo(v)
			}
		}
		if len(pairs) > 0 {
			util.Logf("script: restored %d state entries from db", len(pairs))
		}
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		util.Logf("script: directory %q not found; no scripts loaded", dir)
		return e, nil
	}
	if err != nil {
		return nil, fmt.Errorf("script: read dir %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tengo") {
			continue
		}
		topic := strings.TrimSuffix(entry.Name(), ".tengo")
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("script: read %q: %w", entry.Name(), err)
		}
		compiled, err := e.compile(topic, src)
		if err != nil {
			return nil, fmt.Errorf("script: compile %q: %w", entry.Name(), err)
		}
		e.scripts[topic] = compiled
		util.Logf("script: loaded handler for topic %q", topic)
	}

	return e, nil
}

func (e *Engine) compile(topic string, src []byte) (*tengo.Compiled, error) {
	s := tengo.NewScript(src)

	s.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	if err := s.Add("notify", e.notifyFunc()); err != nil {
		return nil, err
	}
	if err := s.Add("get_state", e.getStateFunc()); err != nil {
		return nil, err
	}
	if err := s.Add("set_state", e.setStateFunc()); err != nil {
		return nil, err
	}
	if err := s.Add("del_state", e.delStateFunc()); err != nil {
		return nil, err
	}
	if err := s.Add("throttle", e.throttleFunc()); err != nil {
		return nil, err
	}

	for _, fn := range builtins.DBFuncs(e.db) {
		if err := s.Add(fn.Name, fn); err != nil {
			return nil, err
		}
	}

	for _, fn := range builtins.GPIOFuncs(e.gpio) {
		if err := s.Add(fn.Name, fn); err != nil {
			return nil, err
		}
	}

	for _, fn := range builtins.All() {
		if err := s.Add(fn.Name, fn); err != nil {
			return nil, err
		}
	}

	for name, val := range builtins.Vars() {
		if err := s.Add(name, val); err != nil {
			return nil, err
		}
	}

	if err := s.Add("payload", map[string]interface{}{}); err != nil {
		return nil, err
	}
	if err := s.Add("topic", topic); err != nil {
		return nil, err
	}

	return s.Compile()
}

func (e *Engine) HandleEvent(topic string, payload map[string]any) bool {
	compiled, ok := e.scripts[topic]
	if !ok {
		return false
	}

	c := compiled.Clone()

	if err := c.Set("payload", payload); err != nil {
		util.Logf("script: set payload for %q: %v", topic, err)
	}
	if err := c.Set("topic", topic); err != nil {
		util.Logf("script: set topic for %q: %v", topic, err)
	}

	if err := c.Run(); err != nil {
		util.Logf("script: run %q: %v", topic, err)
	}

	return true
}

func (e *Engine) notifyFunc() *tengo.UserFunction {
	return &tengo.UserFunction{
		Name: "notify",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 1 {
				return tengo.UndefinedValue, nil
			}
			topicObj, ok := args[0].(*tengo.String)
			if !ok {
				return tengo.UndefinedValue, fmt.Errorf("notify: first argument must be a string")
			}
			topic := topicObj.Value
			payload := map[string]any{}
			if len(args) >= 2 {
				if m, ok := args[1].(*tengo.Map); ok {
					payload = builtins.TengoMapToGo(m)
				}
			}
			e.nc.Notify(topic, payload)
			return tengo.UndefinedValue, nil
		},
	}
}
