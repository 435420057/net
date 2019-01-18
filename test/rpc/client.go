package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"math/rand"
	"sync"
	"time"
)

var (
	CMD_RPC = uint32(1)

	wg = sync.WaitGroup{}

	mtx       = sync.Mutex{}
	data      = []byte{}
	clientNum = 16
	loopNum   = 50000
)

func startRpcClient() {
	defer wg.Done()

	addr := "127.0.0.1:8888"
	client, err := net.NewRpcClient(addr, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
	}

	var reqdata = make([]byte, 256)
	var rspdata []byte
	for i := 0; i < loopNum; i++ {
		rand.Read(reqdata)
		rspdata, err = client.Call(CMD_RPC, reqdata)
		if err != nil || string(reqdata) != string(rspdata) {
			log.Debug("rpc failed: %v", err)
		}
	}
	mtx.Lock()
	mtx.Unlock()
}

func main() {
	t0 := time.Now()
	for i := 0; i < clientNum; i++ {
		wg.Add(1)
		go startRpcClient()
	}
	wg.Wait()
	seconds := time.Since(t0).Seconds()
	log.Debug("total used: %v, request: %d, %d / s", seconds, loopNum*clientNum, int(float64(loopNum*clientNum)/seconds))
}
