package net

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	tcpservers      = make(map[string]*TcpServer)
	tcpserversMutex = sync.Mutex{}

	_client_rm_from_server = "^_*18616!%$"
)

type ITcpServer interface {
	ITcpEngin
	Start(addr string) error
	Stop()
	StopWithTimeout(to time.Duration)
	Serve(addr string, args ...interface{})
	CurrLoad() int32
	MaxLoad() int32
	SetMaxConcurrent(maxLoad int32)
	HandleServerStop(stopHandler func(server ITcpServer))
	Broadcast(msg IMessage)
	BroadcastWithFilter(msg IMessage, filter func(ITcpClient) bool)
}

type TcpServer struct {
	TcpEngin
	tag           string
	running       bool
	clientCount   uint64
	currLoad      int32
	maxLoad       int32
	listener      *net.TCPListener
	stopTimeout   time.Duration
	onStopHandler func(server ITcpServer)
}

func (server *TcpServer) addClient(client ITcpClient) {
	server.Lock()
	server.clients[client] = struct{}{}
	server.Unlock()
	atomic.AddInt32(&server.currLoad, 1)
}

func (server *TcpServer) deleClient(client ITcpClient) {
	server.Lock()
	delete(server.clients, client)
	server.Unlock()
	atomic.AddInt32(&server.currLoad, -1)
}

func (server *TcpServer) Broadcast(msg IMessage) {
	server.Lock()
	for c, _ := range server.clients {
		c.SendMsg(msg)
	}
	server.Unlock()
}

func (server *TcpServer) BroadcastWithFilter(msg IMessage, filter func(ITcpClient) bool) {
	server.Lock()
	for c, _ := range server.clients {
		if filter(c) {
			c.SendMsg(msg)
		}
	}
	server.Unlock()
}

func (server *TcpServer) stopClients() {
	server.Lock()
	defer server.Unlock()

	for client, _ := range server.clients {
		client.CancelOnClose(_client_rm_from_server)
		client.Stop()
	}
}

func (server *TcpServer) startListener(addr string) error {
	defer logDebug("[TcpServer %s] Stopped.", server.tag)
	var (
		err       error
		idx       uint64
		conn      *net.TCPConn
		tcpAddr   *net.TCPAddr
		client    ITcpClient
		file      *os.File
		tempDelay time.Duration
	)

	tcpAddr, err = net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logDebug("[TcpServer %s] ResolveTCPAddr error: %v", server.tag, err)
		return err
	}

	server.listener, err = net.ListenTCP("tcp", tcpAddr)

	if err != nil {
		logDebug("[TcpServer %s] Listening error: %v", server.tag, err)
		return err
	}

	defer server.listener.Close()

	server.running = true

	logDebug("[TcpServer %s] Running on: \"%s\"", server.tag, tcpAddr.String())
	for server.running {
		if conn, err = server.listener.AcceptTCP(); err == nil {
			if server.maxLoad == 0 || atomic.LoadInt32(&server.currLoad) < server.maxLoad {
				if runtime.GOOS == "linux" {
					if file, err = conn.File(); err == nil {
						idx = uint64(file.Fd())
					}
				} else {
					server.clientCount++
					if server.clientCount < 0 {
						server.clientCount = 0
					}
					idx = server.clientCount
				}

				if err = server.OnNewConn(conn); err == nil {
					client = server.CreateClient(idx, conn, server, server.NewCipher())
					server.addClient(client)
					client.OnClose(_client_rm_from_server, server.deleClient)
					client.Start()
					server.OnNewClient(client)
				} else {
					logDebug("[TcpServer %s] init conn error: %v\n", server.tag, err)
				}
			} else {
				conn.Close()
			}
		} else {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				logDebug("[TcpServer %s] Accept error: %v; retrying in %v", server.tag, err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			logDebug("[TcpServer %s] Accept error: %v\n", server.tag, err)
			break
		}
	}
	return err
}

func (server *TcpServer) Start(addr string) error {
	server.Lock()
	running := server.running
	server.running = true
	server.Unlock()

	if !running {
		server.Add(1)
		defer deleteTcpServer(server.tag)
		return server.startListener(addr)
	}
	return fmt.Errorf("server already started")
}

func (server *TcpServer) Stop() {
	server.Lock()
	running := server.running
	server.running = false
	server.Unlock()
	defer handlePanic()

	if !running {
		return
	}

	server.listener.Close()
	server.Done()

	if server.stopTimeout > 0 {
		timer := time.AfterFunc(server.stopTimeout, func() {
			logDebug("[TcpServer %s] Stop Timeout.", server.tag)
			// time.Sleep(time.Second / 10)
			// os.Exit(-1)
		})
		defer timer.Stop()
	}

	logDebug("[TcpServer %s] Stop Waiting...", server.tag)

	server.Wait()

	server.stopClients()

	if server.onStopHandler != nil {
		server.onStopHandler(server)
	}
	time.Sleep(time.Second / 10)
	logDebug("[TcpServer %s] Stop Done.", server.tag)
}

func (server *TcpServer) StopWithTimeout(to time.Duration) {
	server.stopTimeout = to
	server.Stop()
}

func (server *TcpServer) Serve(addr string, args ...interface{}) {
	safeGo(func() {
		server.Start(addr)
	})

	if len(args) > 0 {
		if to, ok := args[0].(time.Duration); ok {
			server.stopTimeout = to
		}
	}

	handleSignal(func(sig os.Signal) {
		if sig == syscall.SIGTERM || sig == syscall.SIGINT {
			server.Stop()
			os.Exit(0)
		}
	})
}

func (server *TcpServer) CurrLoad() int32 {
	return atomic.LoadInt32(&server.currLoad)
}

func (server *TcpServer) MaxLoad() int32 {
	return server.maxLoad
}

func (server *TcpServer) SetMaxConcurrent(maxLoad int32) {
	server.maxLoad = maxLoad
}

//handle message by cmd
func (server *TcpServer) HandleServerStop(stopHandler func(server ITcpServer)) {
	server.onStopHandler = stopHandler
}

func NewTcpServer(tag string) ITcpServer {
	tcpserversMutex.Lock()
	defer tcpserversMutex.Unlock()

	if _, ok := tcpservers[tag]; ok {
		logDebug("NewTcpServer Error: (TcpServer-%s) already exists.", tag)
		return nil
	}

	server := &TcpServer{
		TcpEngin: TcpEngin{
			clients: map[ITcpClient]struct{}{},
			handlerMap: map[uint32]func(ITcpClient, IMessage){
				CmdSetReaIp: func(client ITcpClient, msg IMessage) {
					ip := msg.Body()
					if len(ip) == 4 {
						client.SetRealIp(fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3]))
					} else {
						client.SetRealIp(string(ip))
					}
				},
			},

			sockNoDelay:       _conf_sock_nodelay,
			sockKeepAlive:     _conf_sock_keepalive,
			sendQsize:         _conf_sock_send_q_size,
			sockRecvBufLen:    _conf_sock_recv_buf_len,
			sockSendBufLen:    _conf_sock_send_buf_len,
			sockMaxPackLen:    _conf_sock_pack_max_len,
			sockRecvBlockTime: _conf_sock_recv_block_time,
			sockSendBlockTime: _conf_sock_send_block_time,
			sockKeepaliveTime: _conf_sock_keepalive_time,
		},
		tag: tag,
	}

	tcpservers[tag] = server

	return server
}

func deleteTcpServer(name string) {
	tcpserversMutex.Lock()
	defer tcpserversMutex.Unlock()

	delete(tcpservers, name)
}

func GetTcpServerBytag(name string) (*TcpServer, bool) {
	tcpserversMutex.Lock()
	defer tcpserversMutex.Unlock()
	server, ok := tcpservers[name]
	return server, ok
}
