package media

import (
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3/pkg/media"
	"log"
	"math"
)

type VideoSampler struct {
	depacketizer rtp.Depacketizer
	buffer       []*videoBufferElem
}

func NewVideoSampler() *VideoSampler {
	return &VideoSampler{
		depacketizer: &codecs.VP8Packet{},
	}
}

type videoBufferElem struct {
	id     uint16
	packet *rtp.Packet
	data   []byte
	start  bool
	end    bool
}

func (b *videoBufferElem) isAfter(other *videoBufferElem) bool {
	if b.id < 4096 && other.id > math.MaxUint16-(4096) {
		return true
	} else if b.id > math.MaxUint16-(4096) && other.id < 4096 {
		return true
	}

	return b.id > other.id
}

func (s *VideoSampler) Push(p *rtp.Packet) {
	isStart := s.depacketizer.IsPartitionHead(p.Payload)
	isEnd := s.depacketizer.IsPartitionTail(p.Marker, p.Payload)

	blob, err := s.depacketizer.Unmarshal(p.Payload)
	if err != nil {
		log.Println("depacketizer error", err)
		return
	}

	elem := &videoBufferElem{
		id:     p.SequenceNumber,
		packet: p,
		data:   blob,
		start:  isStart,
		end:    isEnd,
	}

	inserted := false

	for i := 0; i < len(s.buffer); i++ {
		other := s.buffer[i]
		if other.isAfter(elem) {
			s.buffer = append(s.buffer[:i+1], s.buffer[i:]...)
			s.buffer[i] = elem
			inserted = true
			break
		}
	}

	if !inserted {
		s.buffer = append(s.buffer, elem)
	}
}

func (s *VideoSampler) Pop() *media.Sample {
	inPacket := false
	var hasGap bool
	var lastId uint16
	foundPacket := false
	startIndex := 0
	droppedPackets := 0

	for i := 0; i < len(s.buffer); i++ {
		elem := s.buffer[i]

		if elem.start {
			inPacket = true
			hasGap = false
			lastId = elem.id
			startIndex = i
		} else if inPacket {
			if lastId+1 != elem.id {
				hasGap = true
			}

			lastId = elem.id
		}

		if inPacket && elem.end {
			inPacket = false

			if hasGap {
				//log.Println("Packet starting at", startIndex, "is incomplete")
				continue
			} else if startIndex == 0 {
				//log.Println("Found complete packet at start")
				foundPacket = true
			} else {
				keyframe := s.buffer[startIndex].data[0]&0x1 == 0

				if keyframe {
					log.Println("Fast forwarding to complete keyframe")
					s.buffer = s.buffer[startIndex:]
					droppedPackets = startIndex
					foundPacket = true
				}
			}

		}
	}

	if !foundPacket {
		return nil
	}

	var data []byte
	var startElem *videoBufferElem

	i := 0
	for {
		elem := s.buffer[i]

		if i == 0 {
			startElem = elem
			lastId = elem.id

			if !elem.start {
				log.Fatal("Unexpected start state")
			}
		} else if lastId+1 != elem.id {
			log.Fatal("Unexpected missing content")
		}

		lastId = elem.id
		data = append(data, elem.data...)
		i++

		if elem.end {
			break
		}
	}

	s.buffer = s.buffer[i:]

	sample := &media.Sample{
		Data: data,
		//Timestamp:          time.Time{},
		//Duration:           0,
		PacketTimestamp:    startElem.packet.Timestamp,
		PrevDroppedPackets: uint16(droppedPackets),
	}

	return sample
}
