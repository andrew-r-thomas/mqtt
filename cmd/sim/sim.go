package main

import (
	"context"
	"encoding/binary"
	"log"
	"math"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/google/uuid"
)

const clients = 5

var duration = time.Second * 10

func main() {
	var wg sync.WaitGroup
	stampChan := make(chan timeStamp)

	for range clients {
		wg.Add(1)
		go func() {
			time.Sleep(time.Second)
			id := uuid.New().String()
			runClient(duration, id, stampChan)
			wg.Done()
		}()
	}

	resChan := make(chan time.Duration, 1)
	go collectStats(stampChan, resChan)

	wg.Wait()
	close(stampChan)

	avgLatency := <-resChan
	log.Printf("average latency: %v\n", avgLatency)
}

type EventType byte

const (
	MsgSend EventType = iota
	MsgRecv
)

type timeStamp struct {
	t     time.Time
	id    uint32
	eType EventType
}

func runClient(dur time.Duration, id string, times chan<- timeStamp) {
	conn, err := net.Dial("tcp", ":1883")
	if err != nil {
		log.Fatalf("%s: error dialing broker: %v\n", id, err)
	}

	client := paho.NewClient(paho.ClientConfig{
		OnClientError: func(err error) {
			log.Printf("%s: client error: %v\n", id, err)
		},
		OnPublishReceived: []func(
			p paho.PublishReceived,
		) (bool, error){
			func(p paho.PublishReceived) (bool, error) {
				log.Printf("%s: received pub\n", id)
				t := time.Now()
				id := binary.BigEndian.Uint32(p.Packet.Payload[:4])
				times <- timeStamp{
					t:     t,
					eType: MsgRecv,
					id:    id,
				}
				return true, nil
			},
		},
		Conn: conn,
	})

	ctx := context.Background()
	_, err = client.Connect(
		ctx,
		&paho.Connect{
			KeepAlive:    30,
			ClientID:     id,
			CleanStart:   true,
			UsernameFlag: false,
			PasswordFlag: false,
		},
	)

	if err != nil {
		log.Fatalf("%s: error connecting to broker: %v\n", id, err)
	} else {
		log.Printf("%s: connected\n", id)
	}

	_, err = client.Subscribe(
		ctx,
		&paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{
					Topic:   "test",
					QoS:     0,
					NoLocal: false,
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("%s: error subscribing: %v\n", id, err)
	} else {
		log.Printf("%s: subscribed\n", id)
	}

	to := time.After(dur)
	var pid uint32 = 0
	for {
		time.Sleep(time.Second)
		select {
		case <-to:
			err = client.Disconnect(&paho.Disconnect{ReasonCode: 0})
			if err != nil {
				log.Fatalf("%s: error disconnecting: %v\n", id, err)
			} else {
				log.Printf("%s: disconnected\n", id)
			}
			return
		default:
			lat := rand.Float64()
			long := rand.Float64()

			payload := make([]byte, 20)
			binary.BigEndian.PutUint32(payload[:4], pid)
			binary.BigEndian.PutUint64(payload[4:12], math.Float64bits(lat))
			binary.BigEndian.PutUint64(payload[12:], math.Float64bits(long))

			pub := &paho.Publish{
				Topic:   "test",
				QoS:     0,
				Payload: payload,
			}

			sendTime := time.Now()
			_, err := client.Publish(ctx, pub)
			if err != nil {
				log.Fatalf("%s: error publishing: %v\n", id, err)
			} else {
				log.Printf("%s: published message\n", id)
			}

			times <- timeStamp{
				t:     sendTime,
				eType: MsgSend,
				id:    pid,
			}

			pid++
		}
	}
}

func collectStats(stamps <-chan timeStamp, resChan chan<- time.Duration) {
	sents := map[uint32]time.Time{}
	var avgLatency time.Duration
	n := 0

	for stamp := range stamps {
		switch stamp.eType {
		case MsgSend:
			sents[stamp.id] = stamp.t
		case MsgRecv:
			latency := stamp.t.Sub(sents[stamp.id])
			avgLatency += latency
			n++
			delete(sents, stamp.id)
		}
	}

	avgLatency /= time.Duration(n)

	resChan <- avgLatency
}
