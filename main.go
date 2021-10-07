package main

import (
	"qperf-go/client"
	"qperf-go/server"
	"sync"
)

const addr = "localhost:4242"

func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	go server.Run(addr, &wg)

	client.Run(addr)
	wg.Wait()
}
