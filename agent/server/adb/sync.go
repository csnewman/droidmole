package adb

import (
	"encoding/binary"
	"errors"
)

const MaxData = 64 * 1024

const IdListV2 = 'L' | ('I' << 8) | ('S' << 16) | ('2' << 24)
const IdDentV2 = 'D' | ('N' << 8) | ('T' << 16) | ('2' << 24)
const IdDone = 'D' | ('O' << 8) | ('N' << 16) | ('E' << 24)

const IdStatV2 = 'S' | ('T' << 8) | ('A' << 16) | ('2' << 24)
const IdLStatV2 = 'L' | ('S' << 8) | ('T' << 16) | ('2' << 24)

const IdRecvV2 = 'R' | ('C' << 8) | ('V' << 16) | ('2' << 24)
const IdFail = 'F' | ('A' << 8) | ('I' << 16) | ('L' << 24)
const IdData = 'D' | ('A' << 8) | ('T' << 16) | ('A' << 24)

const IdSendV2 = 'S' | ('N' << 8) | ('D' << 16) | ('2' << 24)
const IdOkay = 'O' | ('K' << 8) | ('A' << 16) | ('Y' << 24)

const FileStatSize = 4 * 16

type FileStat struct {
	Dev   uint64
	INo   uint64
	Mode  uint32
	NLink uint32
	UId   uint32
	GId   uint32
	Size  uint64
	ATime int64
	MTime int64
	CTime int64
}

func parseFileStat(data []byte) FileStat {
	return FileStat{
		Dev:   binary.LittleEndian.Uint64(data[0:8]),
		INo:   binary.LittleEndian.Uint64(data[8:16]),
		Mode:  binary.LittleEndian.Uint32(data[16:20]),
		NLink: binary.LittleEndian.Uint32(data[20:24]),
		UId:   binary.LittleEndian.Uint32(data[24:28]),
		GId:   binary.LittleEndian.Uint32(data[28:32]),
		Size:  binary.LittleEndian.Uint64(data[32:40]),
		ATime: int64(binary.LittleEndian.Uint64(data[40:48])),
		MTime: int64(binary.LittleEndian.Uint64(data[48:56])),
		CTime: int64(binary.LittleEndian.Uint64(data[56:64])),
	}
}

type ListDirectoryEntry struct {
	Name      string
	StatError uint32
	Stat      FileStat
}

func (s *systemImpl) ListDirectory(path string) ([]ListDirectoryEntry, error) {
	conn, err := s.OpenEmulator()
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	err = conn.SendCommand([]byte("sync:"))
	if err != nil {
		return nil, err
	}

	// Request listing
	packet := make([]byte, 8+len(path))
	binary.LittleEndian.PutUint32(packet[0:4], uint32(IdListV2))
	binary.LittleEndian.PutUint32(packet[4:8], uint32(len(path)))
	copy(packet[8:], path)

	err = conn.WriteRaw(packet)
	if err != nil {
		return nil, err
	}

	var entries []ListDirectoryEntry

	// Read entries
	for {
		respPacket := make([]byte, FileStatSize+12)
		err = conn.ReadRaw(respPacket)
		if err != nil {
			return nil, err
		}

		id := binary.LittleEndian.Uint32(respPacket[0:4])
		statErr := binary.LittleEndian.Uint32(respPacket[4:8])
		if id == IdDone {
			break
		} else if id != IdDentV2 {
			return nil, errors.New("unexpected response id")
		}

		fs := parseFileStat(respPacket[8 : 8+FileStatSize])

		nameLen := binary.LittleEndian.Uint32(respPacket[8+FileStatSize : 12+FileStatSize])
		name := make([]byte, nameLen)
		err = conn.ReadRaw(name)
		if err != nil {
			return nil, err
		}

		entries = append(entries, ListDirectoryEntry{
			Name:      string(name),
			StatError: statErr,
			Stat:      fs,
		})
	}

	return entries, nil
}

func (s *systemImpl) StatFile(path string, followLinks bool) (uint32, *FileStat, error) {
	conn, err := s.OpenEmulator()
	if err != nil {
		return 0, nil, err
	}

	defer conn.Close()

	err = conn.SendCommand([]byte("sync:"))
	if err != nil {
		return 0, nil, err
	}

	packet := make([]byte, 8+len(path))
	if followLinks {
		binary.LittleEndian.PutUint32(packet[0:4], uint32(IdStatV2))
	} else {
		binary.LittleEndian.PutUint32(packet[0:4], uint32(IdLStatV2))
	}

	binary.LittleEndian.PutUint32(packet[4:8], uint32(len(path)))
	copy(packet[8:], path)

	err = conn.WriteRaw(packet)
	if err != nil {
		return 0, nil, err
	}

	respPacket := make([]byte, FileStatSize+8)
	err = conn.ReadRaw(respPacket)
	if err != nil {
		return 0, nil, err
	}

	id := binary.LittleEndian.Uint32(respPacket[0:4])
	statError := binary.LittleEndian.Uint32(respPacket[4:8])

	if id == IdFail {
		msg := make([]byte, statError)
		err = conn.ReadRaw(msg)
		if err != nil {
			return 0, nil, err
		}

		return 0, nil, errors.New(string(msg))
	} else if id != IdStatV2 && id != IdLStatV2 {
		return 0, nil, errors.New("unexpected response id")
	}

	fs := parseFileStat(respPacket[8 : 8+FileStatSize])
	return statError, &fs, nil
}

type PullFileStream struct {
	conn RawConnection
}

func (s *systemImpl) PullFile(path string) (*PullFileStream, error) {
	conn, err := s.OpenEmulator()
	if err != nil {
		return nil, err
	}

	err = conn.SendCommand([]byte("sync:"))
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Request file
	packet := make([]byte, 16+len(path))
	binary.LittleEndian.PutUint32(packet[0:4], uint32(IdRecvV2))
	binary.LittleEndian.PutUint32(packet[4:8], uint32(len(path)))
	pos := len(path) + 8
	copy(packet[8:pos], path)
	binary.LittleEndian.PutUint32(packet[pos:pos+4], uint32(IdRecvV2))
	binary.LittleEndian.PutUint32(packet[pos+4:pos+8], uint32(0))

	err = conn.WriteRaw(packet)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &PullFileStream{
		conn: conn,
	}, nil
}

func (s *PullFileStream) Recv() ([]byte, error) {
	respPacket := make([]byte, 8)
	err := s.conn.ReadRaw(respPacket)
	if err != nil {
		return nil, err
	}

	id := binary.LittleEndian.Uint32(respPacket[0:4])
	length := binary.LittleEndian.Uint32(respPacket[4:8])

	blob := make([]byte, length)
	err = s.conn.ReadRaw(blob)
	if err != nil {
		return nil, err
	}

	if id == IdFail {
		s.conn.Close()
		return nil, errors.New(string(blob))
	} else if id == IdData {
		return blob, nil
	} else if id == IdDone {
		s.conn.Close()
		return nil, nil
	} else {
		s.conn.Close()
		return nil, errors.New("unexpected resp id")
	}
}

func (s *PullFileStream) Close() error {
	return s.conn.Close()
}

type PushFileStream struct {
	conn RawConnection
}

func (s *systemImpl) PushFile(path string, mode uint32) (*PushFileStream, error) {
	conn, err := s.OpenEmulator()
	if err != nil {
		return nil, err
	}

	err = conn.SendCommand([]byte("sync:"))
	if err != nil {
		conn.Close()
		return nil, err
	}

	packet := make([]byte, 20+len(path))
	binary.LittleEndian.PutUint32(packet[0:4], uint32(IdSendV2))
	binary.LittleEndian.PutUint32(packet[4:8], uint32(len(path)))
	pos := len(path) + 8
	copy(packet[8:pos], path)
	binary.LittleEndian.PutUint32(packet[pos:pos+4], uint32(IdSendV2))
	binary.LittleEndian.PutUint32(packet[pos+4:pos+8], mode)
	binary.LittleEndian.PutUint32(packet[pos+8:pos+12], uint32(0))

	err = conn.WriteRaw(packet)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &PushFileStream{
		conn: conn,
	}, nil
}

func (s *PushFileStream) Send(blob []byte) error {
	if len(blob) > MaxData {
		s.conn.Close()
		return errors.New("send blob over max size")
	}

	packet := make([]byte, 8)
	binary.LittleEndian.PutUint32(packet[0:4], uint32(IdData))
	binary.LittleEndian.PutUint32(packet[4:8], uint32(len(blob)))
	err := s.conn.WriteRaw(packet)
	if err != nil {
		s.conn.Close()
		return err
	}

	err = s.conn.WriteRaw(blob)
	if err != nil {
		s.conn.Close()
		return err
	}

	return nil
}

func (s *PushFileStream) Done(mtime uint32) error {
	defer s.conn.Close()

	// Send "done" packet
	packet := make([]byte, 8)
	binary.LittleEndian.PutUint32(packet[0:4], uint32(IdDone))
	binary.LittleEndian.PutUint32(packet[4:8], mtime)
	err := s.conn.WriteRaw(packet)

	// Read receipt
	respPacket := make([]byte, 8)
	err = s.conn.ReadRaw(respPacket)
	if err != nil {
		return err
	}

	id := binary.LittleEndian.Uint32(respPacket[0:4])
	length := binary.LittleEndian.Uint32(respPacket[4:8])

	blob := make([]byte, length)
	err = s.conn.ReadRaw(blob)
	if err != nil {
		return err
	}

	if id == IdFail {
		// blob contains error message
		return errors.New(string(blob))
	} else if id == IdOkay {
		return nil
	} else {
		return errors.New("unexpected resp id")
	}
}

func (s *PushFileStream) Close() error {
	return s.conn.Close()
}
