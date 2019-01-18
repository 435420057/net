package net

import (
	"sync/atomic"
	"time"
)

type IRpcClient interface {
	ITcpClient
	Call(cmd uint32, data []byte) ([]byte, error)
	CallWithTimeout(cmd uint32, data []byte, timeout time.Duration) ([]byte, error)
}

type rpcsession struct {
	seq  int64
	done chan IMessage
}

type RpcClient struct {
	*TcpClient
	sessionMap map[int64]*rpcsession
}

func (client *RpcClient) removeSession(seq int64) {
	client.Lock()
	delete(client.sessionMap, seq)
	client.Unlock()
}

func (client *RpcClient) Call(cmd uint32, data []byte) ([]byte, error) {
	var session *rpcsession
	client.Lock()
	if client.running {
		session = &rpcsession{
			seq:  atomic.AddInt64(&client.sendSeq, 1),
			done: make(chan IMessage, 1),
		}
		msg := NewRpcMessage(cmd, session.seq, data)
		select {
		case client.chSend <- asyncMessage{msg.Data(), nil}:
			client.sessionMap[session.seq] = session
		default:
			client.parent.OnSendQueueFull(client, msg)
			return nil, ErrRpcClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		return nil, ErrRpcClientIsStopped
	}
	client.Unlock()
	defer client.removeSession(session.seq)
	msg, ok := <-session.done
	if !ok {
		return nil, ErrRpcCallClientError
	}
	return msg.Body(), nil
}

func (client *RpcClient) CallWithTimeout(cmd uint32, data []byte, timeout time.Duration) ([]byte, error) {
	var session *rpcsession
	client.Lock()
	if client.running {
		session = &rpcsession{
			seq:  atomic.AddInt64(&client.sendSeq, 1),
			done: make(chan IMessage, 1),
		}
		msg := NewRpcMessage(cmd, session.seq, data)
		select {
		case client.chSend <- asyncMessage{msg.Data(), nil}:
			client.sessionMap[session.seq] = session
		default:
			client.parent.OnSendQueueFull(client, msg)
			return nil, ErrRpcClientSendQueueIsFull
		}
	} else {
		client.Unlock()
		return nil, ErrRpcClientIsStopped
	}
	client.Unlock()
	select {
	case msg, ok := <-session.done:
		if !ok {
			return nil, ErrRpcCallClientError
		}
		return msg.Body(), nil
	case <-time.After(timeout):
		return nil, ErrRpcCallTimeout
	}
	return nil, ErrRpcCallClientError
}

func NewRpcClient(addr string, onConnected func(ITcpClient)) (IRpcClient, error) {
	engine := NewTcpEngine()
	engine.SetSockRecvBlockTime(_conf_sock_rpc_recv_block_time)
	client, err := NewTcpClient(addr, engine, nil, true, onConnected)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		client.Keepalive(_conf_sock_keepalive_time)
	})
	rpcclient := &RpcClient{client, map[int64]*rpcsession{}}
	rpcclient.OnClose("-", func(ITcpClient) {
		rpcclient.Lock()
		defer rpcclient.Unlock()
		for _, session := range rpcclient.sessionMap {
			close(session.done)
		}
		rpcclient.sessionMap = map[int64]*rpcsession{}
	})

	engine.HandleOnMessage(func(c ITcpClient, msg IMessage) {
		rpcclient.Lock()
		session, ok := rpcclient.sessionMap[msg.RpcSeq()]
		rpcclient.Unlock()
		if ok {
			session.done <- msg
		}
	})

	return rpcclient, nil
}
