package net

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"
)

var (
	inited = false
)

const (
	maxStack  = 20
	separator = "---------------------------------------\n"
)

func Ping(addr string, to time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, to)
	if err == nil {
		defer conn.Close()

	}
	return err
}

func handlePanic() interface{} {
	if err := recover(); err != nil {
		errstr := fmt.Sprintf("%sruntime error: %v\ntraceback:\n", separator, err)

		i := 2
		for {
			pc, file, line, ok := runtime.Caller(i)
			if !ok || i > maxStack {
				break
			}
			errstr += fmt.Sprintf("    stack: %d %v [file: %s] [func: %s] [line: %d]\n", i-1, ok, file, runtime.FuncForPC(pc).Name(), line)
			i++
		}
		errstr += separator

		logDebug(errstr)

		return err
	}
	return nil
}

func safe(cb func()) {
	defer handlePanic()
	cb()
}

func safeGo(cb func()) {
	go func() {
		defer handlePanic()
		cb()
	}()
}

func handleSignal(handler func(sig os.Signal)) {
	if !inited {
		inited = true
		chSignal := make(chan os.Signal, 1)
		//signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
		signal.Notify(chSignal)
		for {
			if sig, ok := <-chSignal; ok {
				logDebug("Recv Signal: %v", sig)

				if handler != nil {
					handler(sig)
				}
			} else {
				return
			}
		}
	}
}
