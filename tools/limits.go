package tools

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type LimitQueeMap struct {
	sync.RWMutex
	LimitQueue map[string][]int64
}

func (l *LimitQueeMap) allow(key string, count uint, timeWindow int64, currentTime int64) bool {
	l.Lock()
	defer l.Unlock()

	if l.LimitQueue == nil {
		l.LimitQueue = make(map[string][]int64)
	}

	queue := l.LimitQueue[key]
	firstValid := 0
	for firstValid < len(queue) && currentTime-queue[firstValid] > timeWindow {
		firstValid++
	}
	queue = queue[firstValid:]
	if uint(len(queue)) >= count {
		l.LimitQueue[key] = queue
		return false
	}

	l.LimitQueue[key] = append(queue, currentTime)
	return true
}

func (l *LimitQueeMap) reset() {
	l.Lock()
	l.LimitQueue = make(map[string][]int64)
	l.Unlock()
}

var LimitQueue = &LimitQueeMap{
	LimitQueue: make(map[string][]int64),
}

func NewLimitQueue() {
	cleanLimitQueue()
}
func cleanLimitQueue() {
	go func() {
		for {
			log.Println("cleanLimitQueue start...")
			LimitQueue.reset()
			now := time.Now()
			// 计算下一个零点
			next := now.Add(time.Hour * 24)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			t := time.NewTimer(next.Sub(now))
			<-t.C
		}
	}()
}

// LimitFreqSingle 单机时间滑动窗口限流法
func LimitFreqSingle(queueName string, count uint, timeWindow int64) bool {
	if !viper.GetBool("rate_limit.enabled") {
		return true
	}
	if count == 0 || timeWindow < 0 {
		return true
	}
	return LimitQueue.allow(queueName, count, timeWindow, time.Now().Unix())
}
