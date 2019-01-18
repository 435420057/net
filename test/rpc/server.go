package main

import (
	"github.com/temprory/net"
	"time"
)

const (
	CMD_RPC = uint32(1)
)

func onRpc(client net.ITcpClient, msg net.IMessage) {
	client.SendMsg(net.NewRpcMessage(CMD_RPC, msg.RpcSeq(), msg.Body()))
}

func main() {
	server := net.NewTcpServer("rpc")
	server.SetMaxConcurrent(1000)
	server.HandleRpc(CMD_RPC, onRpc)
	server.Serve(":8888", time.Second*5)
}
