package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

var ()

const (
	CMD_ECHO = uint32(1)
)

func onEcho(client net.ITcpClient, msg net.IMessage) {
	log.Info("server onEcho recv from %v: %v", client.Conn().RemoteAddr().String(), string(msg.Body()))
	err := client.SendMsg(msg)
	log.Info("server send to%s: %v, %v,", client.Conn().RemoteAddr().String(), string(msg.Body()), err)

}

func main() {
	server := net.NewTcpServer("echo")
	server.Handle(CMD_ECHO, onEcho)
	server.Serve(":8888", time.Second*5)
}
