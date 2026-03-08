package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"gnetClient/protobuf/pbGo"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

const HeaderLen = 12

// =======================
// 协议结构
// =======================

type MessageHead struct {
	Len      uint16
	Flag     uint16
	SN       uint32
	Code     uint16
	Protocol uint16
}

type Message struct {
	Head *MessageHead
	Body []byte
}

// =======================
// 封包
// =======================

func PackMessage(msg *Message) ([]byte, error) {
	buf := new(bytes.Buffer)

	msg.Head.Len = uint16(len(msg.Body))

	// 按网络字节序写入
	if err := binary.Write(buf, binary.BigEndian, msg.Head.Len); err != nil {
		return nil, fmt.Errorf("write len failed: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, msg.Head.Flag); err != nil {
		return nil, fmt.Errorf("write flag failed: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, msg.Head.SN); err != nil {
		return nil, fmt.Errorf("write sn failed: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, msg.Head.Code); err != nil {
		return nil, fmt.Errorf("write code failed: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, msg.Head.Protocol); err != nil {
		return nil, fmt.Errorf("write protocol failed: %w", err)
	}

	if _, err := buf.Write(msg.Body); err != nil {
		return nil, fmt.Errorf("write body failed: %w", err)
	}

	return buf.Bytes(), nil
}

// =======================
// 解包
// =======================

func ReadMessage(conn net.Conn) (*Message, error) {
	headerBuf := make([]byte, HeaderLen)

	if _, err := io.ReadFull(conn, headerBuf); err != nil {
		return nil, fmt.Errorf("read header failed: %w", err)
	}

	reader := bytes.NewReader(headerBuf)
	head := &MessageHead{}

	if err := binary.Read(reader, binary.BigEndian, &head.Len); err != nil {
		return nil, fmt.Errorf("parse len failed: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &head.Flag); err != nil {
		return nil, fmt.Errorf("parse flag failed: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &head.SN); err != nil {
		return nil, fmt.Errorf("parse sn failed: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &head.Code); err != nil {
		return nil, fmt.Errorf("parse code failed: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &head.Protocol); err != nil {
		return nil, fmt.Errorf("parse protocol failed: %w", err)
	}

	body := make([]byte, head.Len)
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	return &Message{
		Head: head,
		Body: body,
	}, nil
}

// =======================
// 消息发送函数
// =======================

func SendLoginReq(conn net.Conn, sn uint32, protocolId uint16) error {
	body, err := proto.Marshal(&pbGo.LoginReq{
		Uuid:  1,
		Token: "大黄",
	})
	if err != nil {
		return fmt.Errorf("marshal login req failed: %w", err)
	}

	msg := &Message{
		Head: &MessageHead{
			Flag:     0,
			SN:       sn,
			Code:     0,
			Protocol: protocolId,
		},
		Body: body,
	}

	data, err := PackMessage(msg)
	if err != nil {
		return fmt.Errorf("pack message failed: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write to conn failed: %w", err)
	}

	fmt.Printf("发送请求: SN=%d, protocolId=%d,\n", sn, protocolId)
	return nil
}

func SendTestRpcReq(conn net.Conn, sn uint32, protocolId uint16) error {
	body, err := proto.Marshal(&pbGo.TestRpcRep{
		Id:   1,
		Name: "test大大",
	})
	if err != nil {
		return fmt.Errorf("marshal test rpc req failed: %w", err)
	}

	msg := &Message{
		Head: &MessageHead{
			Flag:     0,
			SN:       sn,
			Code:     0,
			Protocol: protocolId,
		},
		Body: body,
	}

	data, err := PackMessage(msg)
	if err != nil {
		return fmt.Errorf("pack message failed: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write to conn failed: %w", err)
	}

	fmt.Printf("发送RPC请求: SN=%d, protocolId=%d, \n", sn, protocolId)
	return nil
}

// =======================
// 消息处理器
// =======================

type MessageHandler struct {
	conn net.Conn
	sn   uint32
	mu   sync.Mutex
}

func NewMessageHandler(conn net.Conn) *MessageHandler {
	return &MessageHandler{
		conn: conn,
		sn:   1,
	}
}

func (h *MessageHandler) NextSN() uint32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	sn := h.sn
	h.sn++
	return sn
}

func (h *MessageHandler) HandleMessage(msg *Message) {
	fmt.Printf("\n[收到消息] SN=%d | Protocol=%d | Code=%d | BodyLen=%d\n",
		msg.Head.SN, msg.Head.Protocol, msg.Head.Code, msg.Head.Len)

	switch msg.Head.Protocol {
	case 1: // 登录响应
		var resp pbGo.LoginResp
		if err := proto.Unmarshal(msg.Body, &resp); err != nil {
			fmt.Printf("解析登录响应失败: %v\n", err)
			return
		}
		fmt.Printf("登录响应: UUID=%d, Name=%s\n", resp.Uuid, resp.Name)

	case 101: // TestRpc响应
		var resp pbGo.TestRpcResp
		if err := proto.Unmarshal(msg.Body, &resp); err != nil {
			fmt.Printf("解析RPC响应失败: %v\n", err)
			return
		}
		fmt.Printf("RPC响应: ID=%d, Name=%s\n", resp.Id, resp.Name)

	default:
		fmt.Printf("未知协议: %d\n", msg.Head.Protocol)
	}
}

func main() {
	addr := "127.0.0.1:7002"

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("连接成功: %s\n", addr)

	handler := NewMessageHandler(conn)
	done := make(chan struct{})

	// 启动接收协程
	go func() {
		defer close(done)
		for {
			msg, err := ReadMessage(conn)
			if err != nil {
				if err == io.EOF {
					fmt.Println("\n服务器断开连接")
				} else {
					fmt.Printf("\n读取失败: %v\n", err)
				}
				return
			}
			handler.HandleMessage(msg)
			fmt.Print("> ") // 重新打印提示符
		}
	}()

	// 主循环：处理用户输入
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		select {
		case <-done:
			fmt.Println("连接已关闭，程序退出")
			return
		default:
		}

		fields := strings.Fields(scanner.Text())
		protocolInt, err := strconv.Atoi(fields[0]) // 第一个必须是数字
		switch protocolInt {
		case 1:
			if err = SendLoginReq(conn, handler.NextSN(), uint16(protocolInt)); err != nil {
				fmt.Printf("发送失败: %v\n", err)
			}

		case 101:
			if err := SendTestRpcReq(conn, handler.NextSN(), uint16(protocolInt)); err != nil {
				fmt.Printf("发送失败: %v\n", err)
			}

		default:
			fmt.Printf("未知命令: %d\n", protocolInt)
		}

		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("读取输入错误: %v\n", err)
	}
}
