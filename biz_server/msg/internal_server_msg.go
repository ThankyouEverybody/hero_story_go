package msg

import (
	"bytes"
	"encoding/binary"
)

type InternalServerMsg struct {
	GatewayServerId int32
	SessionId       int32
	UserId          int64
	MsgData         []byte //原始消息

}

// ToByteArray 序列化
func (msg *InternalServerMsg) ToByteArray() []byte {
	buff := bytes.NewBuffer([]byte{})
	_ = binary.Write(buff, binary.BigEndian, msg.GatewayServerId)
	_ = binary.Write(buff, binary.BigEndian, msg.SessionId)
	_ = binary.Write(buff, binary.BigEndian, msg.UserId)
	_ = binary.Write(buff, binary.BigEndian, msg.MsgData)
	return buff.Bytes()
}

// FromByteArray 反序列化
func (msg *InternalServerMsg) FromByteArray(byteArray []byte) {
	if nil == byteArray || len(byteArray) <= 0 {
		return
	}
	buff := bytes.NewBuffer(byteArray)
	_ = binary.Read(buff, binary.BigEndian, &msg.GatewayServerId)
	_ = binary.Read(buff, binary.BigEndian, &msg.SessionId)
	_ = binary.Read(buff, binary.BigEndian, &msg.UserId)
	msg.MsgData = buff.Bytes()
}
