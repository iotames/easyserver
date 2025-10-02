package httpsvr

import (
	"fmt"
	"time"
)

type GlobalData struct {
	Key        string
	Rewritable bool
	Value      interface{}
	CreatedAt  time.Time
}

func (s *EasyServer) SetData(k string, v interface{}) error {
	if s.data == nil {
		s.data = make(map[string]GlobalData)
	}
	vv, ok := s.data[k]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s could not rewritable", k)
	}
	s.data[k] = GlobalData{Key: k, Value: v, CreatedAt: time.Now(), Rewritable: true}
	return nil
}

func (s *EasyServer) SetDataReadonly(k string, v interface{}) error {
	if s.data == nil {
		s.data = make(map[string]GlobalData)
	}
	vv, ok := s.data[k]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s could not rewritable", k)
	}
	s.data[k] = GlobalData{Key: k, Value: v, CreatedAt: time.Now(), Rewritable: false}
	return nil
}

func (s *EasyServer) GetData(k string) GlobalData {
	if s.data == nil {
		return GlobalData{}
	}
	v, ok := s.data[k]
	if ok {
		return v
	}
	return GlobalData{}
}
