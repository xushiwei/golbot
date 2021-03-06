package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/yuin/gopher-lua"
	"layeh.com/gopher-luar"
)

func pushN(L *lua.LState, values ...lua.LValue) {
	for _, v := range values {
		L.Push(v)
	}
}

func getStringField(L *lua.LState, t lua.LValue, key string) (string, bool) {
	lv := L.GetField(t, key)
	if s, ok := lv.(lua.LString); ok {
		return string(s), true
	}
	return "", false
}

func getNumberField(L *lua.LState, t lua.LValue, key string) (float64, bool) {
	lv := L.GetField(t, key)
	if n, ok := lv.(lua.LNumber); ok {
		return float64(n), true
	}
	return 0, false
}

func toCamel(s string) string {
	return strings.Replace(strings.Title(strings.Replace(s, "_", " ", -1)), " ", "", -1)
}

func luaToXml(lvalue lua.LValue) string {
	buf := []string{}
	return strings.Join(_luaToXml(lvalue, buf), " ")
}

func _luaToXml(lvalue lua.LValue, buf []string) []string {
	switch v := lvalue.(type) {
	case *lua.LTable:
		tag := v.RawGetInt(1).String()
		buf = append(buf, fmt.Sprintf("<%s", tag))
		v.ForEach(func(key, value lua.LValue) {
			switch kv := key.(type) {
			case lua.LNumber:
			default:
				buf = append(buf, fmt.Sprintf(" %s=\"%s\"", kv.String(), value.String()))
			}
		})
		buf = append(buf, ">")
		v.ForEach(func(key, value lua.LValue) {
			if kv, ok := key.(lua.LNumber); ok {
				if kv == 1 {
					return
				}
				if s, ok := key.(lua.LString); ok {
					buf = append(buf, s.String())
				} else {
					buf = _luaToXml(value, buf)
				}
			}
		})
		buf = append(buf, fmt.Sprintf("</%s>", tag))
	}
	return buf
}

func proxyLuar(L *lua.LState, tp interface{}, methods func(*lua.LState, string) bool) {
	mt := luar.MT(L, tp)
	newIndexFn := mt.RawGetString("__newindex")
	indexFn := mt.RawGetString("__index")
	mt.RawSetString("__newindex", L.NewFunction(func(L *lua.LState) int {
		pushN(L, newIndexFn, L.Get(1), lua.LString(toCamel(L.CheckString(2))), L.Get(3))
		L.Call(3, 0)
		return 0
	}))

	mt.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		if methods == nil || !methods(L, key) {
			pushN(L, indexFn, L.Get(1), lua.LString(toCamel(key)))
			L.Call(2, 1)
		}
		return 1
	}))
}

func mustDecodeJson(data []byte) map[string]interface{} {
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		panic(err)
	}
	return asObject(result)
}

func asObject(v interface{}) map[string]interface{} {
	return v.(map[string]interface{})
}

func asArray(v interface{}) []interface{} {
	return v.([]interface{})
}

func propertyPath(obj map[string]interface{}, path string) interface{} {
	pathparts := strings.Split(path, ".")
	var cur interface{}
	cur = obj
	for _, p := range pathparts {
		pi := strings.Index(p, "[")
		if pi > 0 {
			pe := strings.Index(p, "]")
			key := p[0:pi]
			index, _ := strconv.Atoi(p[pi+1 : pe])
			cur = (cur.(map[string]interface{}))[key]
			cur = (cur.([]interface{}))[index]
		} else {
			cur = (cur.(map[string]interface{}))[p]
		}
	}
	return cur
}
