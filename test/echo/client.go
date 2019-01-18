package main

import (
	"fmt"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

const (
	CMD_ECHO = uint32(1)
)

func onEcho(client net.ITcpClient, msg net.IMessage) {
	log.Debug("client onEcho recv from %v: %v", client.Conn().RemoteAddr().String(), string(msg.Body()))
}

func onConnected(client net.ITcpClient) {
	for i := 0; i < 20; i++ {
		err := client.SendMsg(net.NewMessage(CMD_ECHO, []byte(fmt.Sprintf("hello %v", i+1))))
		log.Debug("client send to %v: %v, %v", client.Conn().RemoteAddr().String(), fmt.Sprintf("hello %v", i+1), err)
		if err != nil {
			break
		}
		time.Sleep(time.Second)
	}
}

func main() {
	var (
		err        error
		addr       = "127.0.0.1:8888"
		client     net.ITcpClient
		cipher     net.ICipher = nil
		autoReconn             = true
		netengine              = net.NewTcpEngine()
	)

	netengine.Handle(CMD_ECHO, onEcho)

	client, err = net.NewTcpClient(addr, netengine, cipher, autoReconn, onConnected)
	if err != nil {
		log.Debug("NewTcpClient failed: %v, %v", client, err)
	}

	<-make(chan int)
}
