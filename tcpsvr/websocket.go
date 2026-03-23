package tcpsvr

import (
	// "bytes"
	"encoding/binary"
	"errors"
	// "fmt"
)

// WebSocketUnpack 解包客户端发送的 WebSocket 帧，返回 payload、opcode 和错误。
// 要求 data 包含至少一个完整帧，且客户端帧必须带掩码。
func WebSocketUnpack(data []byte) (payload []byte, opcode byte, err error) {
	if len(data) < 2 {
		return nil, 0, errors.New("frame too short")
	}
	firstByte := data[0]
	secondByte := data[1]

	// 解析 opcode
	opcode = firstByte & 0x0F

	// 检查 FIN 位（假设我们只处理完整帧，不支持分片）
	fin := (firstByte & 0x80) != 0
	if !fin {
		return nil, opcode, errors.New("fragmented frames not supported")
	}

	// 客户端帧必须带掩码
	masked := (secondByte & 0x80) != 0
	if !masked {
		return nil, opcode, errors.New("client frame must be masked")
	}

	payloadLen := int(secondByte & 0x7F)
	offset := 2
	if payloadLen == 126 {
		if len(data) < 4 {
			return nil, opcode, errors.New("frame too short for extended length")
		}
		payloadLen = int(binary.BigEndian.Uint16(data[2:4]))
		offset = 4
	} else if payloadLen == 127 {
		if len(data) < 10 {
			return nil, opcode, errors.New("frame too short for extended length 64-bit")
		}
		payloadLen64 := binary.BigEndian.Uint64(data[2:10])
		// 限制最大 2GB，避免内存问题
		if payloadLen64 > 0x7FFFFFFF {
			return nil, opcode, errors.New("payload too large")
		}
		payloadLen = int(payloadLen64)
		offset = 10
	}

	if len(data) < offset+4+payloadLen {
		return nil, opcode, errors.New("incomplete frame")
	}

	maskKey := data[offset : offset+4]
	encoded := data[offset+4 : offset+4+payloadLen]

	// 解除掩码
	payload = make([]byte, payloadLen)
	for i, b := range encoded {
		payload[i] = b ^ maskKey[i%4]
	}
	return payload, opcode, nil
}

// WebSocketPack 将数据打包成 WebSocket 帧（服务端发送，无掩码），默认使用文本帧(0x1)
// 若需发送二进制数据，请使用 WebSocketPackBinary
func WebSocketPack(data []byte) []byte {
	return WebSocketPackWithOpcode(data, 0x1)
}

// WebSocketPackBinary 打包数据为二进制帧(0x2)
func WebSocketPackBinary(data []byte) []byte {
	return WebSocketPackWithOpcode(data, 0x2)
}

// WebSocketPackWithOpcode 允许指定 opcode 打包帧
func WebSocketPackWithOpcode(data []byte, opcode byte) []byte {
	length := len(data)
	// 第一个字节：FIN=1, RSV1-3=0, opcode
	firstByte := byte(0x80 | opcode) // 0x80 = 10000000

	var header []byte
	if length <= 125 {
		header = []byte{firstByte, byte(length)}
	} else if length <= 65535 {
		header = []byte{firstByte, 126, byte(length >> 8), byte(length)}
	} else {
		// 长度超过 65535，使用 8 字节表示
		header = make([]byte, 10)
		header[0] = firstByte
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:], uint64(length))
	}
	return append(header, data...)
}

// // 整形转换成字节
// func IntToBytes(n int, b byte) ([]byte, error) {
// 	switch b {
// 	case 1:
// 		tmp := int8(n)
// 		bytesBuffer := bytes.NewBuffer([]byte{})
// 		binary.Write(bytesBuffer, binary.BigEndian, &tmp)
// 		return bytesBuffer.Bytes(), nil
// 	case 2:
// 		tmp := int16(n)
// 		bytesBuffer := bytes.NewBuffer([]byte{})
// 		binary.Write(bytesBuffer, binary.BigEndian, &tmp)
// 		return bytesBuffer.Bytes(), nil
// 	case 3, 4:
// 		tmp := int32(n)
// 		bytesBuffer := bytes.NewBuffer([]byte{})
// 		binary.Write(bytesBuffer, binary.BigEndian, &tmp)
// 		return bytesBuffer.Bytes(), nil
// 	}
// 	return nil, fmt.Errorf("IntToBytes b param is invaild")
// }

// func BytesCombine(pBytes ...[]byte) []byte {
// 	return bytes.Join(pBytes, []byte(""))
// }

// func WebSocketUnpack(data []byte) []byte {
// 	en_bytes := []byte("")
// 	cn_bytes := make([]int, 0)

// 	v := data[1] & 0x7f
// 	p := 0
// 	switch v {
// 	case 0x7e:
// 		p = 4
// 	case 0x7f:
// 		p = 10
// 	default:
// 		p = 2
// 	}
// 	mask := data[p : p+4]
// 	data_tmp := data[p+4:]
// 	nv := ""
// 	nv_bytes := []byte("")
// 	nv_len := 0

// 	for k, v := range data_tmp {

// 		nv = string(int(v ^ mask[k%4]))
// 		// nv = fmt.Sprintf("%d", int(v^mask[k%4]))
// 		nv_bytes = []byte(nv)
// 		nv_len = len(nv_bytes)
// 		if nv_len == 1 {
// 			en_bytes = BytesCombine(en_bytes, nv_bytes)
// 		} else {
// 			en_bytes = BytesCombine(en_bytes, []byte("%s"))
// 			cn_bytes = append(cn_bytes, int(v^mask[k%4]))
// 		}
// 	}

// 	//处理中文
// 	cn_str := make([]interface{}, 0)
// 	if len(cn_bytes) > 2 {
// 		clen := len(cn_bytes)
// 		count := int(clen / 3)

// 		for i := 0; i < count; i++ {
// 			mm := i * 3

// 			hh := make([]byte, 3)
// 			h1, _ := IntToBytes(cn_bytes[mm], 1)
// 			h2, _ := IntToBytes(cn_bytes[mm+1], 1)
// 			h3, _ := IntToBytes(cn_bytes[mm+2], 1)
// 			hh[0] = h1[0]
// 			hh[1] = h2[0]
// 			hh[2] = h3[0]

// 			cn_str = append(cn_str, string(hh))
// 		}
// 		// TODO string to []byte
// 		new := string(bytes.Replace(en_bytes, []byte("%s%s%s"), []byte("%s"), -1))
// 		return []byte(fmt.Sprintf(new, cn_str...))

// 	}
// 	return en_bytes
// }

// func WebSocketPack(data []byte) []byte {
// 	lenth := len(data)
// 	token := string(0x81)
// 	if lenth < 126 {
// 		token += string(lenth)
// 	}
// 	bb, _ := IntToBytes(0x81, 1)
// 	b0 := bb[0]
// 	b1 := byte(0)
// 	framePos := 0
// 	// fmt.Println("长度", lenth)
// 	switch {
// 	case lenth >= 65536:
// 		writeBuf := make([]byte, 10)
// 		writeBuf[framePos] = b0
// 		writeBuf[framePos+1] = b1 | 127
// 		binary.BigEndian.PutUint64(writeBuf[framePos+2:], uint64(lenth))

// 		return BytesCombine(writeBuf, data)
// 	case lenth > 125:
// 		fmt.Println("》125")
// 		writeBuf := make([]byte, 4)
// 		writeBuf[framePos] = b0
// 		writeBuf[framePos+1] = b1 | 126
// 		binary.BigEndian.PutUint16(writeBuf[framePos+2:], uint16(lenth))
// 		fmt.Println(writeBuf)
// 		return BytesCombine(writeBuf, data)
// 	default:
// 		writeBuf := make([]byte, 2)
// 		writeBuf[framePos] = b0
// 		writeBuf[framePos+1] = b1 | byte(lenth)

// 		return BytesCombine(writeBuf, data)
// 	}
// }
