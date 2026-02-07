// @Summary, an LRU Cache that will be used for the services
package main

import (
	"sync"
)

type Node struct {
	data interface{}
	next *Node
	prev *Node
}

type DoublyLinkedList struct {
	head  *Node
	tail  *Node
	count int
}

type LRU struct {
	capacity uint64
	cache    map[string]*Node
	list     *DoublyLinkedList
	lock     sync.RWMutex
}

func NewLRU(capacity int) *LRU {
	return &LRU{
		capacity: uint64(capacity),
		cache:    make(map[string]*Node),
		list:     &DoublyLinkedList{},
	}
}

func (l *LRU) addToFront(value interface{}) bool {
	return true
}

func (l *LRU) delete(value interface{}) bool {
	return true
}

func (l *LRU) Get(key string) (interface{}, bool) {
	return true, true
}

func (l *LRU) Put(key string, value interface{}) {

}
