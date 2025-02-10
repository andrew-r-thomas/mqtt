package test

import (
	"bytes"
	"context"
	"crypto/rand"
	"log"
	"net"

	"github.com/eclipse/paho.golang/paho"
)

func ConformanceTest(ctx context.Context, addr string) {
	payload := make([]byte, 256)
	rand.Read(payload)

	doneChan := make(chan struct{}, 1)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("error dialing: %v\n", err)
	}

	client := paho.NewClient(
		paho.ClientConfig{
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					log.Printf("got pub\n")
					if !bytes.Equal(pr.Packet.Payload, payload) {
						log.Fatalf("incorrect payload\n")
					} else {
						doneChan <- struct{}{}
					}
					return true, nil
				},
			},
			Conn: conn,
		},
	)
	_, err = client.Connect(
		ctx,
		&paho.Connect{
			ClientID:     "client_id",
			CleanStart:   true,
			KeepAlive:    60,
			UsernameFlag: false,
			PasswordFlag: false,
		},
	)
	if err != nil {
		log.Fatalf("error connecting: %v\n", err)
	}

	_, err = client.Subscribe(
		ctx,
		&paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{
					Topic:   "test/topic",
					QoS:     0,
					NoLocal: false,
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("error subscribing: %v\n", err)
	}

	_, err = client.Publish(
		ctx,
		&paho.Publish{
			QoS:     0,
			Payload: payload,
			Topic:   "test/topic",
		},
	)
	if err != nil {
		log.Fatalf("error publishing: %v\n", err)
	} else {
		log.Printf("published\n")
	}

	<-doneChan
}
