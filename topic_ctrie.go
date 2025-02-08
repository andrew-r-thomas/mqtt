package mqtt

import "sync"

type TopicTrie struct {
	lock  *sync.RWMutex
	nodes []Node
}

type Node struct {
	subs     []string
	children map[string]int
}

// find matching subscribers for a given publish topic
// matches is expected to be empty with some capacity
// and topic should valid and checked for errors
func (t *TopicTrie) FindMatches(topic []string, matches []string) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// TODO:
}

// add a subscription to the tree
// topic should be vaild and checked for errors
func (t *TopicTrie) AddSubscription(topic []string, cid string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	var ok bool
	currNode := 0
	for _, level := range topic {
		currNode, ok = t.nodes[currNode].children[level]
		if !ok {
			t.nodes[currNode].children
			currNode = len(t.nodes)
			child := Node{
				subs:     make([]string, 0, 16),
				children: make(map[string]int, 16),
			}
			t.nodes = append(t.nodes, child)
		}
	}
}
