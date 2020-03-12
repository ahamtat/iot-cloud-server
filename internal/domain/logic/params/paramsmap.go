package params

import "sync"

type GuardedParamsMap struct {
	mx     sync.RWMutex
	params map[string]interface{}
}

func NewGuardedParamsMap() *GuardedParamsMap {
	return &GuardedParamsMap{
		mx:     sync.RWMutex{},
		params: make(map[string]interface{}),
	}
}

func (m *GuardedParamsMap) Add(key string, value interface{}) {
	m.mx.Lock()
	m.params[key] = value
	m.mx.Unlock()
}

func (m *GuardedParamsMap) Get(key string) (interface{}, bool) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	value, ok := m.params[key]
	return value, ok
}

func (m *GuardedParamsMap) Remove(key string) {
	m.mx.Lock()
	delete(m.params, key)
	m.mx.Unlock()
}
