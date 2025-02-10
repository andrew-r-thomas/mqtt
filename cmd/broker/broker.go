package main

import (
	"fmt"

	"github.com/andrew-r-thomas/mqtt"
)

const addr = ":1883"

func main() {
	s := mqtt.NewServer()
	err := s.Start(addr)
	fmt.Printf("broker stopped: %v\n", err)
}
