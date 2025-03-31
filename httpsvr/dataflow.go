package httpsvr

import (
	"fmt"
	"sync"
	"time"
)

type DataFlow struct {
	contextMap map[string]GlobalData
	lock       *sync.RWMutex
}

// NewDataFlow 新建数据流传递对象
func NewDataFlow() *DataFlow {
	s := &DataFlow{
		contextMap: make(map[string]GlobalData),
		lock:       &sync.RWMutex{},
	}
	s.SetDataReadonly("startat", time.Now())
	return s
}

func (c DataFlow) GetDataKeys() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	var keys []string
	for k := range c.contextMap {
		keys = append(keys, k)
	}
	return keys
}

// SetData 设置map数据
func (c *DataFlow) SetData(key string, value interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	vv, ok := c.contextMap[key]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s could not rewritable", key)
	}
	c.contextMap[key] = GlobalData{Key: key, Value: value, CreatedAt: time.Now(), Rewritable: true}
	return nil
}

func (c *DataFlow) SetDataReadonly(key string, value interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	vv, ok := c.contextMap[key]
	if ok && !vv.Rewritable {
		// 已存在数据不可被重写覆盖
		return fmt.Errorf("the data with key:%s could not rewritable", key)
	}
	c.contextMap[key] = GlobalData{Key: key, Value: value, CreatedAt: time.Now(), Rewritable: false}
	return nil
}

func (c *DataFlow) GetStr(key string) string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if v, ok := c.contextMap[key]; ok {
		return v.Value.(string)
	}
	return ""
}

// Get 获取GlobalData数据
func (c *DataFlow) GetData(key string) GlobalData {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if v, ok := c.contextMap[key]; ok {
		return v
	}
	return GlobalData{}
}

func (c DataFlow) GetStartAt() time.Time {
	return c.GetData("startat").CreatedAt
}
