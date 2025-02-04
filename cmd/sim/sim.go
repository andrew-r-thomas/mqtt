package main

import (
	"context"
	"encoding/binary"
	"log"
	"math"
	"math/rand/v2"
	"net"
	"time"

	"github.com/eclipse/paho.golang/paho"
)

const clients = 3

func main() {
	for range clients {
		go runClient()
	}
}

func runClient() {
	conn, err := net.Dial("tcp", ":1883")
	if err != nil {
		log.Fatalf("error dialing broker: %v\n", err)
	}

	client := paho.NewClient(paho.ClientConfig{
		OnPublishReceived: []func(
			p paho.PublishReceived,
		) (bool, error){
			func(p paho.PublishReceived) (bool, error) {
				lat_i := binary.BigEndian.Uint64(
					p.Packet.Payload[:8],
				)
				long_i := binary.BigEndian.Uint64(
					p.Packet.Payload[8:],
				)
				lat := math.Float64frombits(lat_i)
				long := math.Float64frombits(long_i)
				log.Printf("recieved lat is: %f\n", lat)
				log.Printf("recieved long is: %f\n", long)
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
			ClientID:     "this is a test",
			CleanStart:   true,
			UsernameFlag: false,
			PasswordFlag: false,
		},
	)

	if err != nil {
		log.Fatalf("error connecting to broker: %v\n", err)
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
		log.Fatalf("error subscribing: %v\n", err)
	}

	for {
		lat := rand.Float64()
		long := rand.Float64()
		log.Printf("sent lat is: %f\n", lat)
		log.Printf("sent long is: %f\n", long)

		payload := make([]byte, 16)
		binary.BigEndian.PutUint64(payload[:8], math.Float64bits(lat))
		binary.BigEndian.PutUint64(payload[8:], math.Float64bits(long))

		_, err := client.Publish(
			ctx,
			&paho.Publish{
				Topic:   "test",
				QoS:     0,
				Payload: payload,
			},
		)
		if err != nil {
			log.Fatalf("error publishing: %v\n", err)
		}

		time.Sleep(time.Second)
	}
}
