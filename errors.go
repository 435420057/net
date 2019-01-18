package net

import (
	"errors"
	"fmt"
)

var (
	ErrTcpClientIsStopped       = errors.New("tcp client is stopped")
	ErrTcpClientSendQueueIsFull = errors.New("tcp client's send queue is full")

	ErrRpcClientIsStopped       = errors.New("rpc client is stopped")
	ErrRpcClientSendQueueIsFull = errors.New("rpc client's send queue is full")
	ErrRpcCallTimeout           = errors.New("rpc call timeout")
	ErrRpcCallClientError       = errors.New("rpc client error")

	_error_invalid_checksum = errors.New("invalid checksum")

	_error_reserved_cmd_ping       = fmt.Errorf("cmd %d is reserved for ping", CmdSetReaIp)
	_error_reserved_cmd_set_realip = fmt.Errorf("cmd %d is reserved for set client's real ip", CmdSetReaIp)
)
