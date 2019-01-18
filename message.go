package net

import (
	"encoding/binary"
	"strconv"
	"strings"
)

type asyncMessage struct {
	data []byte
	cb   func(ITcpClient, error)
}

const (
	_message_head_len int = 16

	_bodylen_idx_begin int = 0
	_bodylen_idx_end   int = 4

	_cmd_idx_begin int = 4
	_cmd_idx_end   int = 8

	_ext_idx_begin int = 8
	_ext_idx_end   int = 16

	CmdPing     = uint32(0)
	CmdSetReaIp = uint32(0xFFFFFFFF)
)

type IMessage interface {
	HeadLen() uint32
	BodyLen() uint32

	Cmd() uint32
	SetCmd(cmd uint32)

	Ext() uint64
	SetExt(ext uint64)

	RpcSeq() int64
	SetRpcSeq(seq int64)

	Data() []byte
	SetData(data []byte)

	RawData() []byte
	SetRawData(rawData []byte)

	Body() []byte
	SetBody(body []byte)

	Encrypt(seq int64, key uint32, cipher ICipher) []byte
	Decrypt(seq int64, key uint32, cipher ICipher) ([]byte, error)
}

type Message struct {
	data    []byte
	rawData []byte
}

func (msg *Message) HeadLen() uint32 {
	return uint32(_message_head_len)
}

func (msg *Message) BodyLen() uint32 {
	return uint32(binary.LittleEndian.Uint32(msg.data[_bodylen_idx_begin:_bodylen_idx_end]))
}

func (msg *Message) Cmd() uint32 {
	return binary.LittleEndian.Uint32(msg.data[_cmd_idx_begin:_cmd_idx_end])
}

func (msg *Message) SetCmd(cmd uint32) {
	binary.LittleEndian.PutUint32(msg.data[_cmd_idx_begin:_cmd_idx_end], cmd)
}

func (msg *Message) Ext() uint64 {
	return binary.LittleEndian.Uint64(msg.data[_ext_idx_begin:_ext_idx_end])
}

func (msg *Message) SetExt(ext uint64) {
	binary.LittleEndian.PutUint64(msg.data[_ext_idx_begin:_ext_idx_end], ext)
}

func (msg *Message) RpcSeq() int64 {
	return int64(binary.LittleEndian.Uint64(msg.data[_ext_idx_begin:_ext_idx_end]))
}

func (msg *Message) SetRpcSeq(seq int64) {
	binary.LittleEndian.PutUint64(msg.data[_ext_idx_begin:_ext_idx_end], uint64(seq))
}

func (msg *Message) Data() []byte {
	return msg.data
}

func (msg *Message) SetData(data []byte) {
	msg.data = data
}

func (msg *Message) RawData() []byte {
	return msg.rawData
}

func (msg *Message) SetRawData(rawData []byte) {
	msg.rawData = rawData
}

func (msg *Message) Body() []byte {
	return msg.data[_message_head_len:]
}

func (msg *Message) SetBody(data []byte) {
	needLen := _message_head_len + len(data) - len(msg.data)
	if needLen > 0 {
		msg.data = append(msg.data, make([]byte, needLen)...)
	} else if needLen < 0 {
		msg.data = msg.data[:_message_head_len+len(data)]
	}
	copy(msg.data[_message_head_len:], data)
	binary.LittleEndian.PutUint32(msg.data[_bodylen_idx_begin:_bodylen_idx_end], uint32(len(data)))
}

func (msg *Message) Encrypt(seq int64, key uint32, cipher ICipher) []byte {
	if cipher != nil {
		msg.rawData = cipher.Encrypt(seq, key, msg.data)
	} else {
		msg.rawData = msg.data
	}
	return msg.rawData
}

func (msg *Message) Decrypt(seq int64, key uint32, cipher ICipher) ([]byte, error) {
	var err error
	if cipher != nil {
		msg.data, err = cipher.Decrypt(seq, key, msg.rawData)
	} else {
		msg.data = msg.rawData
	}
	return msg.data, err
}

func NewMessage(cmd uint32, data []byte) *Message {
	msg := &Message{
		data: make([]byte, len(data)+_message_head_len),
	}
	binary.LittleEndian.PutUint32(msg.data[_cmd_idx_begin:_cmd_idx_end], cmd)
	binary.LittleEndian.PutUint32(msg.data[_bodylen_idx_begin:_bodylen_idx_end], uint32(len(data)))
	if len(data) > 0 {
		copy(msg.data[_message_head_len:], data)
	}
	return msg
}

func NewRpcMessage(cmd uint32, seq int64, data []byte) *Message {
	msg := &Message{
		data: make([]byte, len(data)+_message_head_len),
	}
	binary.LittleEndian.PutUint32(msg.data[_cmd_idx_begin:_cmd_idx_end], cmd)
	binary.LittleEndian.PutUint64(msg.data[_ext_idx_begin:_ext_idx_end], uint64(seq))
	binary.LittleEndian.PutUint32(msg.data[_bodylen_idx_begin:_bodylen_idx_end], uint32(len(data)))
	if len(data) > 0 {
		copy(msg.data[_message_head_len:], data)
	}
	return msg
}

func RealIpMsg(ip string) IMessage {
	ret := strings.Split(ip, ".")
	if len(ret) == 4 {
		var err error
		var ipv int
		var data = make([]byte, 4)
		for i, v := range ret {
			ipv, err = strconv.Atoi(v)
			if err != nil || ipv > 255 {
				return nil
			}
			data[i] = byte(ipv)
		}
		return NewMessage(CmdSetReaIp, data)
	}
	return NewMessage(CmdSetReaIp, []byte(ip))
}

func PingMsg() IMessage {
	return NewMessage(CmdPing, nil)
}
