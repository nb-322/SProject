package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
)

func writeFull(conn net.Conn, buf []byte) error {
	written := 0
	for written < len(buf) {
		n, err := conn.Write(buf[written:])
		if err != nil {
			return err
		}
		written += n
	}
	return nil
}

func ReliableSend(conn net.Conn, msg Message) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	length := uint32(len(jsonData))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)
	_, err = conn.Write(lengthBytes)
	if err != nil {
		return err
	}

	written := 0
	for written < len(jsonData) {
		n, err := conn.Write(jsonData[written:])
		if err != nil {
			return err
		}
		written += n
	}

	return nil
}

func ReliableReceive(conn net.Conn) (Message, error) {
	lengthBytes := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBytes)
	if err != nil {
		return Message{}, err
	}
	length := binary.BigEndian.Uint32(lengthBytes)
	const maxMessageSize = 32 * 1024 * 1024
	if length > maxMessageSize {
		return Message{}, fmt.Errorf("message too large: %d bytes", length)
	}
	dataBytes := make([]byte, length)
	_, err = io.ReadFull(conn, dataBytes)
	if err != nil {
		return Message{}, err
	}
	var msg Message
	err = json.Unmarshal(dataBytes, &msg)
	if err != nil {
		return Message{}, err
	}
	return msg, nil
}

// SendFile отправляет файл по соединению с per-chunk ACK.
// После каждого чанка ждём 1 байт подтверждения от получателя —
// это не даёт bore.pub буфер переполниться при больших файлах.
func SendFile(conn net.Conn, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	if size == 0 {
		return fmt.Errorf("file %s is empty (0 bytes)", filename)
	}

	sizeBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBuf, uint64(size))
	if _, err = conn.Write(sizeBuf); err != nil {
		return err
	}

	fmt.Printf("[protocol] SendFile: sending %s (%d bytes)\n", filename, size)
	buf := make([]byte, 32*1024)
	ackBuf := make([]byte, 1)
	totalSent := int64(0)
	chunk := 0

	for totalSent < size {
		chunk++
		toRead := int64(len(buf))
		if totalSent+toRead > size {
			toRead = size - totalSent
		}

		n, err := file.Read(buf[:toRead])
		if err != nil && err != io.EOF {
			return fmt.Errorf("SendFile read chunk %d: %w", chunk, err)
		}
		if n == 0 {
			break
		}

		if err = writeFull(conn, buf[:n]); err != nil {
			return fmt.Errorf("SendFile write chunk %d: %w", chunk, err)
		}

		if _, err = io.ReadFull(conn, ackBuf); err != nil {
			return fmt.Errorf("SendFile ack chunk %d: %w", chunk, err)
		}
		if ackBuf[0] != 0x01 {
			return fmt.Errorf("SendFile: bad ack byte 0x%02x on chunk %d", ackBuf[0], chunk)
		}

		totalSent += int64(n)
		fmt.Printf("[protocol] SendFile: chunk %d sent+acked (%d/%d bytes)\n", chunk, totalSent, size)
	}

	if totalSent != size {
		return fmt.Errorf("SendFile: incomplete: sent %d of %d", totalSent, size)
	}
	fmt.Printf("[protocol] SendFile: finished %s (%d bytes, %d chunks)\n", filename, totalSent, chunk)
	return nil
}

func ReceiveFile(conn net.Conn, filename string) error {
	sizeBuf := make([]byte, 8)
	if _, err := io.ReadFull(conn, sizeBuf); err != nil {
		return fmt.Errorf("ReceiveFile: read size: %w", err)
	}
	size := int64(binary.LittleEndian.Uint64(sizeBuf))
	fmt.Printf("[protocol] ReceiveFile: receiving %s (%d bytes)\n", filename, size)

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("ReceiveFile: open %s: %w", filename, err)
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	totalReceived := int64(0)
	chunk := 0

	for totalReceived < size {
		chunk++
		toRead := int64(len(buf))
		if totalReceived+toRead > size {
			toRead = size - totalReceived
		}

		n, err := io.ReadFull(conn, buf[:toRead])
		if err != nil {
			return fmt.Errorf("ReceiveFile: read chunk %d: %w", chunk, err)
		}

		written, err := file.Write(buf[:n])
		if err != nil {
			return fmt.Errorf("ReceiveFile: write chunk %d: %w", chunk, err)
		}
		if written != n {
			return fmt.Errorf("ReceiveFile: partial write chunk %d: %d/%d", chunk, written, n)
		}

		// Отправляем ACK отправителю
		if _, err = conn.Write([]byte{0x01}); err != nil {
			return fmt.Errorf("ReceiveFile: send ack chunk %d: %w", chunk, err)
		}

		totalReceived += int64(n)
		fmt.Printf("[protocol] ReceiveFile: chunk %d ok (%d/%d bytes)\n", chunk, totalReceived, size)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("ReceiveFile: sync: %w", err)
	}

	fmt.Printf("[protocol] ReceiveFile: finished %s (%d bytes, %d chunks)\n", filename, totalReceived, chunk)
	return nil
}
