package dbus

import "fmt"

var defaultcbus = New()

func New() *dbus {
	return &dbus{
		data: make(map[interface{}]interface{}),
	}
}

type dbus struct {
	data map[interface{}]interface{}
}

func (p *dbus) Register(k, v interface{}) error {
	if _, ok := p.data[k]; ok {
		return fmt.Errorf("%v already registed", k)
	}

	p.data[k] = v
	return nil
}

func (p *dbus) MustRegister(k, v interface{}) {
	if _, ok := p.data[k]; ok {
		panic(fmt.Errorf("%v already registed", k))
	}

	p.data[k] = v
}

func (p *dbus) Get(k interface{}) (interface{}, bool) {
	v, ok := p.data[k]
	return v, ok
}

func (p *dbus) MustGet(k interface{}) interface{} {
	v, ok := p.data[k]
	if !ok {
		panic(fmt.Sprintf("%v not found", k))
	}
	return v
}

func Register(k, v interface{}) error {
	return defaultcbus.Register(k, v)
}

func Get(k interface{}) (interface{}, bool) {
	return defaultcbus.Get(k)
}

func MustRegister(k, v interface{}) {
	defaultcbus.MustRegister(k, v)
}

func MustGet(k interface{}) interface{} {
	return defaultcbus.MustGet(k)
}
