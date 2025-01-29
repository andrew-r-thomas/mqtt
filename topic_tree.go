/*

TODO:
  - $SYS topics
  - shared subscriptions, starting with round robin load balancing,
    and maybe using $SYS messages for picking a strategy
  - decide if we want to use other '$' prefixed topics, or just not support them

*/

package mqtt

import (
	"log"
	"strings"
)

type TopicTree struct {
	nodes  []topicNode
	cidMap map[string]chan<- []byte

	pubChan    <-chan PubMsg
	subChan    <-chan SubMsg
	unSubChan  <-chan UnSubMsg
	addCliChan <-chan AddCliMsg
	remCliChan <-chan RemCliMsg
}

type topicNode struct {
	subs     []string
	children map[string]int
}

type SubMsg struct {
	TopicFilters [][]string
	ClientId     string
}
type UnSubMsg struct {
	ClientId     string
	TopicFilters [][]string
}
type PubMsg struct {
	Topic string
	Msg   []byte
}
type AddCliMsg struct {
	ClientId string
	Sender   chan<- []byte
}
type RemCliMsg struct {
	ClientId string
}

// these channels should be buffered,
// everything you pass to these channels is assumed to be valid input
// and should be parsed and checked before you send to here, this is because
// passing errors back along these channels is difficult to do and pretty messy
func NewTopicTree(
	pubChan <-chan PubMsg,
	subChan <-chan SubMsg,
	addCliChan <-chan AddCliMsg,
	unSubChan <-chan UnSubMsg,
	remCliChan <-chan RemCliMsg,
) TopicTree {
	return TopicTree{
		nodes: []topicNode{{
			subs:     []string{},
			children: map[string]int{},
		}},
		pubChan:    pubChan,
		subChan:    subChan,
		addCliChan: addCliChan,
		remCliChan: remCliChan,
		unSubChan:  unSubChan,
		cidMap:     make(map[string]chan<- []byte),
	}
}

func (t *TopicTree) Start() {
	for {
		select {
		case conn := <-t.addCliChan:
			t.cidMap[conn.ClientId] = conn.Sender
		case sub := <-t.subChan:
			t.handleSub(sub)
		case pub := <-t.pubChan:
			t.handlePub(pub)
		case unsub := <-t.unSubChan:
			t.handleUnsub(unsub)
		}
	}
}

func (t *TopicTree) handleSub(sub SubMsg) {
	for _, filter := range sub.TopicFilters {
		currNode := &t.nodes[0]
		for _, level := range filter {
			child, ok := currNode.children[level]
			if !ok {
				child = len(t.nodes)
				t.nodes = append(
					t.nodes,
					topicNode{
						subs:     []string{},
						children: map[string]int{},
					},
				)
				currNode.children[level] = child
			}
			currNode = &t.nodes[child]
		}
		log.Printf("adding %s to %v\n", sub.ClientId, currNode)
		currNode.subs = append(currNode.subs, sub.ClientId)
	}
}

// this is the function we want to optimize our datastructure for
func (t *TopicTree) handlePub(pub PubMsg) {
	log.Printf("got a pub in tt\n")
	levels := strings.Split(pub.Topic, "/")
	currNodes := []int{0}
	for _, l := range levels {
		for i, n := range currNodes {
			node := t.nodes[n]
			log.Printf("level: %s\n", l)
			log.Printf("node: %v\n", node)
			wildHash, ok := node.children["#"]
			if ok {
				for _, s := range t.nodes[wildHash].subs {
					t.cidMap[s] <- pub.Msg
				}
			}
			wildPlus, ok := node.children["+"]
			if ok {
				currNodes = append(currNodes, wildPlus)
			}

			level, ok := node.children[l]
			if ok {
				log.Printf("setting currNodes[%d] to %d\n", i, level)
				currNodes[i] = level
			} else {
				currNodes[i] = currNodes[len(currNodes)-1]
				currNodes = currNodes[:len(currNodes)-1]
			}
		}
	}

	for _, c := range currNodes {
		log.Printf("sending to subs of %d\n", c)
		for _, s := range t.nodes[c].subs {
			log.Printf("sending pub to %s\n", s)
			t.cidMap[s] <- pub.Msg
		}
	}
}

func (t *TopicTree) handleUnsub(unsub UnSubMsg) {
	for _, filter := range unsub.TopicFilters {
		currNode := &t.nodes[0]
		for _, level := range filter {
			child, ok := currNode.children[level]
			if !ok {
				break
			}
			currNode = &t.nodes[child]
		}
		for i, s := range currNode.subs {
			if s == unsub.ClientId {
				currNode.subs[i] = currNode.subs[len(currNode.subs)-1]
				currNode.subs = currNode.subs[:len(currNode.subs)-1]
				break
			}
		}
	}
}
