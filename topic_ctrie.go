package mqtt

import (
	"log"
	"sync"
)

type TopicTrie struct {
	lock  sync.RWMutex
	nodes []node
}

type node struct {
	subs     []string
	children map[string]int
}

func NewTopicTrie() TopicTrie {
	return TopicTrie{
		lock: sync.RWMutex{},
		nodes: []node{
			// root
			{
				subs:     make([]string, 0, 16),
				children: make(map[string]int, 16),
			},
		},
	}
}

// find matching subscribers for a given publish topic
// matches is expected to be empty with some capacity
// and topic should valid and checked for errors
func (t *TopicTrie) FindMatches(topic []string) (matches []string) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// TODO:
	// - might want some kind of set to check if we've added a match already
	// (for situations where multiple subs for a client match the topic)

	currNodes := []int{0}
	for _, level := range topic {
		for i, node := range currNodes {
			child, ok := t.nodes[node].children[level]
			if ok {
				currNodes[i] = child
			} else {
				currNodes[i] = currNodes[len(currNodes)-1]
				currNodes = currNodes[:len(currNodes)-1]
			}

			wildHash, ok := t.nodes[node].children["#"]
			if ok {
				matches = append(
					matches,
					t.nodes[wildHash].subs...,
				)
			}

			plusHash, ok := t.nodes[node].children["+"]
			if ok {
				currNodes = append(currNodes, plusHash)
			}
		}
	}

	for _, node := range currNodes {
		for _, match := range t.nodes[node].subs {
			log.Printf("found a match! %s\n", match)
			matches = append(matches, match)
		}
	}

	return
}

// add a subscription to the tree
// topic should be vaild and checked for errors
func (t *TopicTrie) AddSubscription(topic []string, cid string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	currNode := &t.nodes[0] // root
	for _, level := range topic {
		child, ok := currNode.children[level]
		if !ok {
			child = len(t.nodes)
			t.nodes = append(
				t.nodes,
				node{
					subs:     make([]string, 0, 16),
					children: make(map[string]int, 16),
				},
			)
			currNode.children[level] = child
		}
		currNode = &t.nodes[child]
	}

	currNode.subs = append(currNode.subs, cid)
}

// PERF: this is slow as fuck (maybe)
func (t *TopicTrie) RemoveSubs(cid string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, n := range t.nodes {
		for i, s := range n.subs {
			if s == cid {
				n.subs[i] = n.subs[len(n.subs)-1]
				n.subs = n.subs[:len(n.subs)-1]
			}
		}
	}
}
