package websocket

import (
	"encoding/binary"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/reflect/protoreflect"
	"hero_story/biz_server/msg"
	"hero_story/comm/log"
	"time"
)

const oneSecondMilli = 1000
const limitOnceByByteCount = 64 * 1024
const limitByPacketCountPerSecond = 16

// CmdContextImpl Implemented on MyCmdContext
type CmdContextImpl struct {
	userId       int64
	clientIpAddr string
	Conn         *websocket.Conn
	sendMsgQueue chan *protoreflect.ProtoMessage
	SessionId    int32
}

// BindUserId 绑定用户
func (ctx *CmdContextImpl) BindUserId(val int64) {
	ctx.userId = val
}

// GetUserId 获取用户
func (ctx *CmdContextImpl) GetUserId() int64 {
	return ctx.userId
}

// GetClientIpAddr 获取客户端ip
func (ctx *CmdContextImpl) GetClientIpAddr() string {
	return ctx.clientIpAddr
}

// Write 写入消息
func (ctx *CmdContextImpl) Write(msgObj protoreflect.ProtoMessage) {
	if nil == msgObj || ctx.Conn == nil || ctx.sendMsgQueue == nil {
		return
	}
	ctx.sendMsgQueue <- &msgObj

}

// SendError 错误消息
func (ctx *CmdContextImpl) SendError(errorCode int, errorInfo string) {

}

// Disconnect 断开连接
func (ctx *CmdContextImpl) Disconnect() {
	if ctx == nil || ctx.Conn == nil {

		return
	}
	_ = ctx.Conn.Close()
}

// LoopSendMsg 发送消息
func (ctx *CmdContextImpl) LoopSendMsg() {
	if ctx.sendMsgQueue != nil {
		return
	}
	ctx.sendMsgQueue = make(chan *protoreflect.ProtoMessage, 1024)
	go func() {
		for {
			msgObj := <-ctx.sendMsgQueue
			if msgObj == nil {
				continue
			}
			byteArray, err := msg.Encode(msgObj)
			if err != nil {
				log.Error("[websocket] Encode msg error: %v", err)
				return
			}
			if err := ctx.Conn.WriteMessage(websocket.BinaryMessage, byteArray); err != nil {
				log.Error("[websocket] WriteMessage error: %+v", err)
			}
		}
	}()

}

// LoopReadMsg 读取消息
func (ctx *CmdContextImpl) LoopReadMsg() {
	if nil == ctx.Conn {
		return
	}
	ctx.Conn.SetReadLimit(limitOnceByByteCount)
	//limit the count of packet
	t0 := int64(0)
	counter := 0
	//创建游戏服务器连接
	bizServerConn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:12345/websocket", nil)
	if nil != err {
		log.Error("[websocket] Dial error: %+v", err)
		return
	}
	// region 循环读取游戏服发来的消息,转发给客户端
	go func() {
		for {
			//读取游戏服务器返回的数据
			msgType, msgData, err := bizServerConn.ReadMessage()
			if err != nil {
				log.Error("从服务器读取消息失败: %+v", err)
			}
			//ctx.Conn 网关服务器到游戏客户端连接
			if err = ctx.Conn.WriteMessage(msgType, msgData); nil != err {
				log.Error("网关服务器到发送消息失败: %+v", err)
			}
		}
	}()
	//endregion

	// region 循环读取游戏客户端发了的消息,转发给游戏服
	for {
		// 接收游戏客户端消息
		messageType, msgData, err := ctx.Conn.ReadMessage()
		if err != nil {
			log.Error("websocket readMessage error: %v ", err)
			return
		}
		log.Info("msgData: %v", msgData)

		t1 := time.Now().UnixMilli()
		if (t1 - t0) > oneSecondMilli {
			t0 = t1
			counter = 0
		}

		if counter >= limitByPacketCountPerSecond {
			log.Error("消息过于频繁, 拒绝处理, userId: %v, clientIp: %s", ctx.GetUserId(), ctx.GetClientIpAddr())
			return
		}
		counter++
		func() {
			defer func() {
				if e := recover(); e != nil {
					log.Error("解析消息出错：%+v", e)
				}
			}()

			msgCode := binary.BigEndian.Uint16(msgData[2:4])
			message, err := msg.Decode(msgData[4:], int16(msgCode))
			if err != nil {
				log.Error("message message msgCode: %d, err: %+v", msgCode, err)
				return
			}
			log.Info("收到客户端消息 并转发,msgCode: %d, message Name: %v", msgCode, message.Descriptor().Name())
			//向socket服务端发送消息
			//转发给游戏服
			if err = bizServerConn.WriteMessage(messageType, msgData); nil != err {
				log.Error("转发消息失败, err: %+v", err)
			}
		}()

	}
	// endregion
}