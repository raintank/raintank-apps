package message

import (
	"fmt"
	"reflect"
)

type Handler struct {
	Func reflect.Value
	body bool
}

func NewHandler(f interface{}) (*Handler, error) {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		return nil, fmt.Errorf("f is not func")
	}
	ft := fv.Type()
	h := &Handler{
		Func: fv,
	}
	if ft.NumIn() == 0 {
		h.body = false
	}
	if ft.NumIn() == 1 {
		h.body = true
	}
	if ft.NumIn() > 1 {
		return nil, fmt.Errorf("handler func only supports 1 arg.")
	}
	return h, nil
}

func (h *Handler) Call(body []byte) {
	a := make([]reflect.Value, 0)
	if h.body {
		a = append(a, reflect.ValueOf(body))
	}
	h.Func.Call(a)
}
