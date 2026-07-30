package ws

import (
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

type ChatRoom struct {
	sync.RWMutex
	members map[string][]*User
}

// NewRoom 创建用户集合
func NewRoom() *ChatRoom {
	r := &ChatRoom{
		members: make(map[string][]*User),
	}
	r.cleanRoom()
	return r
}

// 清理用户集合
func (r *ChatRoom) cleanRoom() {
	go func() {
		for {
			now := time.Now()
			// 计算下一个零点
			next := now.Add(time.Hour * 24)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			t := time.NewTimer(next.Sub(now))
			<-t.C
			log.Println("cleanRoom start...")
			r.Lock()
			r.members = make(map[string][]*User)
			r.Unlock()
		}
	}()
}

// SendMessageToRoom 发送消息给集合
func (r *ChatRoom) SendMessageToRoom(roomId string, msg []byte) {
	members, _ := r.GetMembers(roomId)
	failed := make(map[*User]struct{})
	for _, member := range members {
		if err := writeUserMessage(member, websocket.TextMessage, msg); err != nil {
			failed[member] = struct{}{}
		}
	}
	r.removeFailedMembers(roomId, failed)
}

// GetMembers 获取用户集合
func (r *ChatRoom) GetMembers(key string) ([]*User, bool) {
	r.RLock()
	value, ok := r.members[key]
	result := append([]*User(nil), value...)
	r.RUnlock()
	return result, ok
}

// SetMembers 设置用户集合
func (r *ChatRoom) SetMembers(key string, value []*User) {
	r.Lock()
	r.members[key] = value
	r.Unlock()
}

// 添加用户
func (r *ChatRoom) addMember(key string, user *User) {
	r.Lock()
	r.members[key] = append(r.members[key], user)
	r.Unlock()
}

// 移除用户
func (r *ChatRoom) removeMember(key string, userId string) {
	r.Lock()
	defer r.Unlock()
	members := r.members[key]
	newMembers := make([]*User, 0)
	for _, member := range members {
		if member.Id != userId {
			newMembers = append(newMembers, member)
		}
	}
	r.members[key] = newMembers
}

func (r *ChatRoom) removeFailedMembers(key string, failed map[*User]struct{}) {
	if len(failed) == 0 {
		return
	}
	r.Lock()
	defer r.Unlock()
	members := r.members[key]
	active := make([]*User, 0, len(members))
	for _, member := range members {
		if _, ok := failed[member]; !ok {
			active = append(active, member)
		}
	}
	r.members[key] = active
}
