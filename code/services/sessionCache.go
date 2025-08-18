package services


import (
	"start-feishubot/services/types" // 使用通用类型
	"sync"
)

type SessionCache struct {
	sessions map[string][]types.Message
	lock     sync.Mutex
}

func NewSessionCache() *SessionCache {
	return &SessionCache{
		sessions: make(map[string][]types.Message),
	}
}

func (c *SessionCache) GetMsg(sessionId string) []types.Message {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.sessions[sessionId]
}

func (c *SessionCache) SetMsg(sessionId string, msg []types.Message) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.sessions[sessionId] = msg
}
关键
