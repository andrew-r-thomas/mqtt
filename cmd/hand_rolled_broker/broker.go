package main

import (
	"fmt"

	"github.com/andrew-r-thomas/mqtt"
)

const addr = ":1883"

func main() {
	s := mqtt.NewServer(addr)
	err := s.Start()
	fmt.Printf("broker stopped: %v\n", err)
}
