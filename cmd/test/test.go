package main

import (
	"context"

	"github.com/andrew-r-thomas/mqtt/test"
)

func main() {
	ctx := context.Background()
	test.ConformanceTest(ctx, ":1883")
}
