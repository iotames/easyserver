package httpsvr

import (
	"fmt"
	"time"
)

type GlobalData struct {
	Key                  string
	Rewritable           bool
	Value                any
	CreatedAt, UpdatedAt time.Time
}

func (s *EasyServer) SetData(k string, v any) error {
	// if s.data == nil {
	// 	s.data = make(map[string]GlobalData)
	// }
	s.lock.Lock()
	defer s.lock.Unlock()

	vv, ok := s.data[k]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s could not rewritable", k)
	}
	ntime := time.Now()
	if ok {
		vv.Value = v
	} else {
		vv = GlobalData{Key: k, Value: v, CreatedAt: ntime, Rewritable: true}
	}
	vv.UpdatedAt = ntime
	s.data[k] = vv
	return nil
}

func (s *EasyServer) SetDataReadonly(k string, v any) error {
	// if s.data == nil {
	// 	s.data = make(map[string]GlobalData)
	// }
	s.lock.Lock()
	defer s.lock.Unlock()

	vv, ok := s.data[k]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s. has exist and could not rewritable", k)
	}
	if ok && vv.Rewritable {
		// 已存在的数据，且已被设置为可被重写覆盖
		return fmt.Errorf("the data with key:%s. has already be set rewritable", k)
	}
	ntime := time.Now()
	s.data[k] = GlobalData{Key: k, Value: v, CreatedAt: ntime, UpdatedAt: ntime, Rewritable: false}
	return nil
}

func (s *EasyServer) GetData(k string) GlobalData {
	// if s.data == nil {
	// 	return GlobalData{}
	// }
	s.lock.RLock()
	defer s.lock.RUnlock()
	v, ok := s.data[k]
	if ok {
		return v
	}
	return GlobalData{}
}
