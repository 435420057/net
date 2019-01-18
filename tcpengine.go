package net

import (
	"io"
	"net"
	"sync"
	"time"
)

type ITcpEngin interface {
	//on new connect callback
	OnNewConn(conn *net.TCPConn) error
	//setting on new connect callback
	HandleNewConn(func(conn *net.TCPConn) error)

	//on new connect callback
	CreateClient(idx uint64, conn *net.TCPConn, parent ITcpEngin, cipher ICipher) ITcpClient

	HandleCreateClient(createClient func(idx uint64, conn *net.TCPConn, parent ITcpEngin, cipher ICipher) ITcpClient)

	//on new connect callback
	OnNewClient(client ITcpClient)
	//setting on new connect callback
	HandleNewClient(onNewClient func(client ITcpClient))

	//on disconnected callback
	NewCipher() ICipher
	//setting on disconnected callback
	HandleNewCipher(func() ICipher)

	//on disconnected callback
	OnDisconnected(client ITcpClient)
	//setting on disconnected callback
	HandleDisconnected(func(client ITcpClient))

	//on send queue is full callback
	OnSendQueueFull(ITcpClient, interface{})
	//setting on disconnected callback
	HandleSendQueueFull(func(ITcpClient, interface{}))

	Send(client ITcpClient, data []byte) error
	HandleSend(func(client ITcpClient, data []byte) error)

	//message router
	RecvMsg(client ITcpClient) IMessage
	//setting message router
	HandleRecv(func(client ITcpClient) IMessage)

	OnMessage(client ITcpClient, msg IMessage)
	HandleOnMessage(onMsg func(client ITcpClient, msg IMessage))

	//handle message by cmd
	Handle(cmd uint32, handler func(client ITcpClient, msg IMessage))
	HandleRpc(cmd uint32, handler func(client ITcpClient, msg IMessage))

	SockNoDelay() bool
	SetSockNoDelay(nodelay bool)

	SockKeepAlive() bool
	SetSockKeepAlive(keepalive bool)

	SockKeepaliveTime() time.Duration
	SetSockKeepaliveTime(keepaliveTime time.Duration)

	SendQueueSize() int
	SetSendQueueSize(size int)

	SockRecvBufLen() int
	SetSockRecvBufLen(recvBufLen int)

	SockSendBufLen() int
	SetSockSendBufLen(sendBufLen int)

	SockMaxPackLen() int
	SetSockMaxPackLen(maxPackLen int)

	SockRecvBlockTime() time.Duration
	SetSockRecvBlockTime(recvBlockTime time.Duration)

	SockSendBlockTime() time.Duration
	SetSockSendBlockTime(sendBlockTime time.Duration)

	BroadCast(msg IMessage)
}

type TcpEngin struct {
	sync.RWMutex
	sync.WaitGroup

	clients    map[ITcpClient]struct{}
	handlerMap map[uint32]func(ITcpClient, IMessage)

	sockNoDelay       bool
	sockKeepAlive     bool
	sendQsize         int
	sockRecvBufLen    int
	sockSendBufLen    int
	sockMaxPackLen    int
	sockKeepaliveTime time.Duration
	sockRecvBlockTime time.Duration
	sockSendBlockTime time.Duration

	onNewConnHandler      func(conn *net.TCPConn) error
	createClientHandler   func(idx uint64, conn *net.TCPConn, parent ITcpEngin, cipher ICipher) ITcpClient
	onNewClientHandler    func(client ITcpClient)
	newCipherHandler      func() ICipher
	onDisconnectedHandler func(client ITcpClient)
	sendQueueFullHandler  func(ITcpClient, interface{})
	recvHandler           func(client ITcpClient) IMessage
	sendHandler           func(client ITcpClient, data []byte) error
	onMsgHandler          func(client ITcpClient, msg IMessage)
}

func (engine *TcpEngin) OnNewConn(conn *net.TCPConn) error {
	if engine.onNewConnHandler != nil {
		return engine.onNewConnHandler(conn)
	}

	var err error
	if err = conn.SetNoDelay(engine.sockNoDelay); err != nil {
		logDebug("SetNoDelay Error: %v.", err)
		goto ErrExit
	}

	if err = conn.SetKeepAlive(engine.sockKeepAlive); err != nil {
		logDebug("SetKeepAlive Error: %v.", err)
		goto ErrExit
	}

	if engine.sockKeepAlive {
		if err = conn.SetKeepAlivePeriod(engine.sockKeepaliveTime); err != nil {
			logDebug("SetKeepAlivePeriod Error: %v.", err)
			goto ErrExit
		}
	}

	if err = conn.SetReadBuffer(engine.sockRecvBufLen); err != nil {
		logDebug("SetReadBuffer Error: %v.", err)
		goto ErrExit
	}
	if err = conn.SetWriteBuffer(engine.sockSendBufLen); err != nil {
		logDebug("SetWriteBuffer Error: %v.", err)
		goto ErrExit
	}

	return nil

ErrExit:
	conn.Close()
	return err
}

func (engine *TcpEngin) HandleNewConn(onNewConn func(conn *net.TCPConn) error) {
	engine.onNewConnHandler = onNewConn
}

//on new connect callback
func (engine *TcpEngin) CreateClient(idx uint64, conn *net.TCPConn, parent ITcpEngin, cipher ICipher) ITcpClient {
	if engine.createClientHandler != nil {
		return engine.createClientHandler(idx, conn, parent, cipher)
	}
	return createTcpClient(idx, conn, parent, cipher)
}

//setting on new connect callback
func (engine *TcpEngin) HandleCreateClient(createClient func(idx uint64, conn *net.TCPConn, parent ITcpEngin, cipher ICipher) ITcpClient) {
	engine.createClientHandler = createClient
}

//on new connect callback
func (engine *TcpEngin) OnNewClient(client ITcpClient) {
	if engine.onNewClientHandler != nil {
		engine.onNewClientHandler(client)
	}
}

//setting on new connect callback
func (engine *TcpEngin) HandleNewClient(onNewClient func(client ITcpClient)) {
	engine.onNewClientHandler = onNewClient
}

//on disconnected callback
func (engine *TcpEngin) NewCipher() ICipher {
	if engine.newCipherHandler != nil {
		return engine.newCipherHandler()
	}
	return nil
}

//setting on disconnected callback
func (engine *TcpEngin) HandleNewCipher(newCipher func() ICipher) {
	engine.newCipherHandler = newCipher
}

//on disconnected callback
func (engine *TcpEngin) OnDisconnected(client ITcpClient) {
	if engine.onDisconnectedHandler != nil {
		engine.onDisconnectedHandler(client)
	}
}

//setting on disconnected callback
func (engine *TcpEngin) HandleDisconnected(onDisconnected func(client ITcpClient)) {
	engine.onDisconnectedHandler = onDisconnected
}

//message router
func (engine *TcpEngin) RecvMsg(client ITcpClient) IMessage {
	// defer handlePanic()

	if engine.recvHandler != nil {
		return engine.recvHandler(client)
	}

	pkt := struct {
		err        error
		msg        *Message
		readLen    int
		dataLen    int
		remoteAddr string
	}{
		err: nil,
		msg: &Message{
			data: make([]byte, _message_head_len),
		},
		readLen:    0,
		dataLen:    0,
		remoteAddr: client.Conn().RemoteAddr().String(),
	}

	if pkt.err = client.Conn().SetReadDeadline(time.Now().Add(engine.sockRecvBlockTime)); pkt.err != nil {
		logDebug("%s RecvMsg SetReadDeadline Err: %v.", client.Conn().RemoteAddr().String(), pkt.err)
		goto Exit
	}

	pkt.readLen, pkt.err = io.ReadFull(client.Conn(), pkt.msg.data)
	if pkt.err != nil || pkt.readLen < _message_head_len {
		logDebug("%s RecvMsg Read Head Err: %v, readLen: %d.", client.Conn().RemoteAddr().String(), pkt.err, pkt.readLen)
		goto Exit
	}

	pkt.dataLen = int(pkt.msg.BodyLen())

	if pkt.dataLen > 0 {
		if pkt.dataLen+_message_head_len > engine.sockMaxPackLen {
			logDebug("%s RecvMsg Read Body Err: Msg Len(%d) > MAXPACK_LEN(%d)", client.Conn().RemoteAddr().String(), pkt.dataLen+_message_head_len, engine.sockMaxPackLen)
			goto Exit
		}

		if pkt.err = client.Conn().SetReadDeadline(time.Now().Add(engine.sockRecvBlockTime)); pkt.err != nil {
			logDebug("%s RecvMsg SetReadDeadline Err: %v.", client.Conn().RemoteAddr().String(), pkt.err)
			goto Exit
		}

		pkt.msg.data = append(pkt.msg.data, make([]byte, pkt.dataLen)...)
		pkt.readLen, pkt.err = io.ReadFull(client.Conn(), pkt.msg.data[_message_head_len:])
		if pkt.err != nil {
			logDebug("%s RecvMsg Read Body Err: %v", client.Conn().RemoteAddr().String(), pkt.err)
			goto Exit
		}
	}

	pkt.msg.rawData = pkt.msg.data
	pkt.msg.data = nil
	if _, pkt.err = pkt.msg.Decrypt(client.RecvSeq(), client.RecvKey(), client.Cipher()); pkt.err != nil {
		logDebug("%s RecvMsg Decrypt Err: %v", client.Conn().RemoteAddr().String(), pkt.err)
		goto Exit
	}

	return pkt.msg

Exit:
	return nil
}

//setting message router
func (engine *TcpEngin) HandleRecv(recver func(client ITcpClient) IMessage) {
	engine.recvHandler = recver
}

func (engine *TcpEngin) OnSendQueueFull(client ITcpClient, msg interface{}) {
	if engine.sendQueueFullHandler != nil {
		engine.sendQueueFullHandler(client, msg)
	}
}

func (engine *TcpEngin) HandleSendQueueFull(cb func(ITcpClient, interface{})) {
	engine.sendQueueFullHandler = cb
}

func (engine *TcpEngin) Send(client ITcpClient, data []byte) error {
	if engine.sendHandler != nil {
		return engine.sendHandler(client, data)
	}
	err := client.Conn().SetWriteDeadline(time.Now().Add(engine.sockSendBlockTime))
	if err != nil {
		logDebug("%s Send SetReadDeadline Err: %v", client.Conn().RemoteAddr().String(), err)
		client.Stop()
		return err
	}

	_, err = client.Conn().Write(data)
	if err != nil {
		logDebug("%s Send Write Err: %v", client.Conn().RemoteAddr().String(), err)
		client.Stop()
	}
	return err
}

func (engine *TcpEngin) HandleSend(sender func(client ITcpClient, data []byte) error) {
	engine.sendHandler = sender
}

func (engine *TcpEngin) OnMessage(client ITcpClient, msg IMessage) {
	if engine.onMsgHandler != nil {
		engine.onMsgHandler(client, msg)
		return
	}

	cmd := msg.Cmd()
	if cmd == CmdPing {
		client.SendMsg(msg)
		return
	}

	if handler, ok := engine.handlerMap[cmd]; ok {
		engine.Add(1)
		defer engine.Done()
		defer handlePanic()
		handler(client, msg)
	} else {
		logDebug("no handler for cmd %d", msg.Cmd())
	}
}

func (engine *TcpEngin) HandleOnMessage(onMsg func(client ITcpClient, msg IMessage)) {
	engine.onMsgHandler = onMsg
}

//handle message by cmd
func (engine *TcpEngin) Handle(cmd uint32, handler func(client ITcpClient, msg IMessage)) {
	if cmd == CmdPing {
		panic(_error_reserved_cmd_ping)
	}
	if cmd == CmdSetReaIp {
		panic(_error_reserved_cmd_set_realip)
	}
	engine.handlerMap[cmd] = handler
}

//handle rpc message by cmd
func (engine *TcpEngin) HandleRpc(cmd uint32, handler func(client ITcpClient, msg IMessage)) {
	engine.Handle(cmd, handler)
}

func (engine *TcpEngin) SockNoDelay() bool {
	return engine.sockNoDelay
}

func (engine *TcpEngin) SetSockNoDelay(nodelay bool) {
	engine.sockNoDelay = nodelay
}

func (engine *TcpEngin) SockKeepAlive() bool {
	return engine.sockKeepAlive
}

func (engine *TcpEngin) SetSockKeepAlive(keepalive bool) {
	engine.sockKeepAlive = keepalive
}

func (engine *TcpEngin) SockKeepaliveTime() time.Duration {
	return engine.sockKeepaliveTime
}

func (engine *TcpEngin) SetSockKeepaliveTime(keepaliveTime time.Duration) {
	engine.sockKeepaliveTime = keepaliveTime
}

func (engine *TcpEngin) SockRecvBufLen() int {
	return engine.sockRecvBufLen
}

func (engine *TcpEngin) SendQueueSize() int {
	return engine.sendQsize
}

func (engine *TcpEngin) SetSendQueueSize(size int) {
	engine.sendQsize = size
}

func (engine *TcpEngin) SetSockRecvBufLen(recvBufLen int) {
	engine.sockRecvBufLen = recvBufLen
}

func (engine *TcpEngin) SockSendBufLen() int {
	return engine.sockSendBufLen
}

func (engine *TcpEngin) SetSockSendBufLen(sendBufLen int) {
	engine.sockSendBufLen = sendBufLen
}

func (engine *TcpEngin) SockRecvBlockTime() time.Duration {
	return engine.sockRecvBlockTime
}

func (engine *TcpEngin) SetSockRecvBlockTime(recvBlockTime time.Duration) {
	engine.sockRecvBlockTime = recvBlockTime
}

func (engine *TcpEngin) SockSendBlockTime() time.Duration {
	return engine.sockSendBlockTime
}

func (engine *TcpEngin) SetSockSendBlockTime(sendBlockTime time.Duration) {
	engine.sockSendBlockTime = sendBlockTime
}

func (engine *TcpEngin) SockMaxPackLen() int {
	return engine.sockMaxPackLen
}

func (engine *TcpEngin) SetSockMaxPackLen(maxPackLen int) {
	engine.sockMaxPackLen = maxPackLen
}

func (engine *TcpEngin) BroadCast(msg IMessage) {
	engine.Lock()
	defer engine.Unlock()
	for c, _ := range engine.clients {
		c.SendMsg(msg)
	}
}

func NewTcpEngine() ITcpEngin {
	return &TcpEngin{
		clients:    map[ITcpClient]struct{}{},
		handlerMap: map[uint32]func(ITcpClient, IMessage){},

		sockNoDelay:       _conf_sock_nodelay,
		sockKeepAlive:     _conf_sock_keepalive,
		sendQsize:         _conf_sock_send_q_size,
		sockRecvBufLen:    _conf_sock_recv_buf_len,
		sockSendBufLen:    _conf_sock_send_buf_len,
		sockMaxPackLen:    _conf_sock_pack_max_len,
		sockRecvBlockTime: _conf_sock_recv_block_time,
		sockSendBlockTime: _conf_sock_send_block_time,
		sockKeepaliveTime: _conf_sock_keepalive_time,
	}
}
