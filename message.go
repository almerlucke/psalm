package psalm

import (
	"fmt"
	"strconv"

	"golang.org/x/exp/constraints"
)

type bang struct{ Bang string }

var Bang = &bang{Bang: "bang"}

func IsBang(value any) bool {
	if value == Bang {
		return true
	}

	if msg, ok := MsgVal(value); ok {
		if _, ok = msg["bang"]; ok {
			return true
		}
	}

	return false
}

func MsgVal(msg any) (Message, bool) {
	switch v := msg.(type) {
	case Message:
		return v, true
	case map[string]any:
		return v, true
	}
	return nil, false
}

func FltVal[T constraints.Float](obj any) T {
	switch v := obj.(type) {
	case T:
		return v
	case bool:
		if v {
			return 1
		}
		return 0
	case float32:
		return T(v)
	case float64:
		return T(v)
	case int:
		return T(v)
	case int8:
		return T(v)
	case int16:
		return T(v)
	case int32:
		return T(v)
	case int64:
		return T(v)
	case uint:
		return T(v)
	case uint8:
		return T(v)
	case uint16:
		return T(v)
	case uint32:
		return T(v)
	case uint64:
		return T(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return T(f)
	default:
		return 0.0
	}
}

func IntVal[T constraints.Integer](obj any) T {
	switch v := obj.(type) {
	case T:
		return v
	case bool:
		if v {
			return 1
		}
		return 0
	case float32:
		return T(v)
	case float64:
		return T(v)
	case int:
		return T(v)
	case int8:
		return T(v)
	case int16:
		return T(v)
	case int32:
		return T(v)
	case int64:
		return T(v)
	case uint:
		return T(v)
	case uint8:
		return T(v)
	case uint16:
		return T(v)
	case uint32:
		return T(v)
	case uint64:
		return T(v)
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return T(i)
	default:
		return 0
	}
}

func BoolVal(obj any) bool {
	switch v := obj.(type) {
	case bool:
		return v
	case float32:
		return v != 0.0
	case float64:
		return v != 0.0
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	default:
		return false
	}
}

func StrVal(obj any) string {
	switch v := obj.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

type Message map[string]any

func (m Message) MustMsg(key string) Message {
	v, _ := m.Msg(key)
	return v
}

func (m Message) Msg(key string) (Message, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}

	return MsgVal(v)
}

func (m Message) MustFltVal(key string) float64 {
	v, _ := m.FltVal(key)
	return v
}

func (m Message) FltVal(key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}

	return FltVal[float64](v), true
}

func (m Message) IfFltVal(key string, f func(v float64)) {
	if v, ok := m[key]; ok {
		f(FltVal[float64](v))
	}
}

func (m Message) MustIntVal(key string) int {
	v, _ := m.IntVal(key)
	return v
}

func (m Message) IntVal(key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}

	return IntVal[int](v), true
}

func (m Message) IfIntVal(key string, f func(v int)) {
	if v, ok := m[key]; ok {
		f(IntVal[int](v))
	}
}

func (m Message) MustBoolVal(key string) bool {
	v, _ := m.BoolVal(key)
	return v
}

func (m Message) BoolVal(key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}

	return BoolVal(v), true
}

func (m Message) IfBoolVal(key string, f func(v bool)) {
	if v, ok := m[key]; ok {
		f(BoolVal(v))
	}
}

func (m Message) MustStrVal(key string) string {
	v, _ := m.StrVal(key)
	return v
}

func (m Message) StrVal(key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}

	return StrVal(v), true
}

func (m Message) IfStrVal(key string, f func(v string)) {
	if v, ok := m[key]; ok {
		f(StrVal(v))
	}
}

func (m Message) MustTickVal(key string) CtrlTick {
	v, _ := m.TickVal(key)
	return v
}

func (m Message) TickVal(key string) (tick CtrlTick, ok bool) {
	tick, ok = m[key].(CtrlTick)
	return
}

func (m Message) IfTickVal(key string, f func(v CtrlTick)) {
	if tick, ok := m[key].(CtrlTick); ok {
		f(tick)
	}
}
