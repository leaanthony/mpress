// Package media inspects video files without external tools so builds and
// checks can tell whether every browser can play them.
package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
)

// Info describes the first video track of a file.
type Info struct {
	Container string // "mp4" or "webm"
	Codec     string // "h264", "hevc", "av1", "vp9", "vp8", "unknown"
	Width     int
	Height    int
}

// ProbeFile reads only container metadata, never the media payload.
func ProbeFile(path string) (Info, error) {
	file, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return Info{}, err
	}
	return Probe(file, stat.Size())
}

func Probe(r io.ReaderAt, size int64) (Info, error) {
	head := make([]byte, 16)
	n, _ := r.ReadAt(head, 0)
	head = head[:n]
	switch {
	case len(head) >= 4 && bytes.Equal(head[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}):
		return probeWebM(r, size)
	case len(head) >= 8 && string(head[4:8]) == "ftyp":
		return probeMP4(r, size)
	}
	return Info{}, errors.New("unrecognised video container")
}

// Playback says which major browsers can decode a codec.
type Playback struct {
	Universal bool   // plays in every current browser, including iPad Safari
	Note      string // human explanation when it is not universal
}

func (i Info) Playback() Playback {
	switch i.Codec {
	case "h264":
		return Playback{Universal: true}
	case "hevc":
		return Playback{Note: "HEVC plays in Safari and Edge but not in Firefox or most Chrome installs"}
	case "av1":
		return Playback{Note: "AV1 needs hardware decoding in Safari (iPad/iPhone with A17 Pro or newer, Mac with M3 or newer)"}
	case "vp9":
		return Playback{Note: "VP9 is not supported by Safari on older iPads and iPhones"}
	case "vp8":
		return Playback{Note: "VP8 is not supported by older Safari"}
	}
	return Playback{Note: "unknown codec"}
}

type box struct {
	kind       string
	start, end int64 // payload range
}

func readBoxes(r io.ReaderAt, from, to int64) ([]box, error) {
	var boxes []box
	for from+8 <= to {
		var h [16]byte
		if _, err := r.ReadAt(h[:8], from); err != nil {
			return boxes, err
		}
		length := int64(binary.BigEndian.Uint32(h[:4]))
		header := int64(8)
		switch length {
		case 0:
			length = to - from
		case 1:
			if _, err := r.ReadAt(h[8:16], from+8); err != nil {
				return boxes, err
			}
			length = int64(binary.BigEndian.Uint64(h[8:16]))
			header = 16
		}
		if length < header || from+length > to {
			return boxes, errors.New("corrupt box")
		}
		boxes = append(boxes, box{string(h[4:8]), from + header, from + length})
		from += length
	}
	return boxes, nil
}

func find(r io.ReaderAt, parent box, kind string) (box, bool) {
	boxes, _ := readBoxes(r, parent.start, parent.end)
	for _, b := range boxes {
		if b.kind == kind {
			return b, true
		}
	}
	return box{}, false
}

func probeMP4(r io.ReaderAt, size int64) (Info, error) {
	top, err := readBoxes(r, 0, size)
	var moov box
	found := false
	for _, b := range top {
		if b.kind == "moov" {
			moov, found = b, true
		}
	}
	if !found {
		if err != nil {
			return Info{}, err
		}
		return Info{}, errors.New("mp4 has no moov box")
	}
	traks, _ := readBoxes(r, moov.start, moov.end)
	for _, trak := range traks {
		if trak.kind != "trak" {
			continue
		}
		mdia, ok := find(r, trak, "mdia")
		if !ok {
			continue
		}
		hdlr, ok := find(r, mdia, "hdlr")
		if !ok {
			continue
		}
		var handler [4]byte
		if _, err := r.ReadAt(handler[:], hdlr.start+8); err != nil || string(handler[:]) != "vide" {
			continue
		}
		info := Info{Container: "mp4", Codec: "unknown"}
		minf, _ := find(r, mdia, "minf")
		stbl, _ := find(r, minf, "stbl")
		stsd, ok := find(r, stbl, "stsd")
		if !ok || stsd.end-stsd.start < 16 {
			return info, nil
		}
		// stsd: version/flags(4) count(4) then the first sample entry.
		entries, _ := readBoxes(r, stsd.start+8, stsd.end)
		if len(entries) > 0 {
			info.Codec = mp4Codec(entries[0].kind)
			var visual [28]byte
			if _, err := r.ReadAt(visual[:], entries[0].start); err == nil {
				info.Width = int(binary.BigEndian.Uint16(visual[24:26]))
				info.Height = int(binary.BigEndian.Uint16(visual[26:28]))
			}
		}
		return info, nil
	}
	return Info{}, errors.New("mp4 has no video track")
}

func mp4Codec(fourcc string) string {
	switch fourcc {
	case "avc1", "avc3":
		return "h264"
	case "hvc1", "hev1", "dvh1", "dvhe":
		return "hevc"
	case "av01":
		return "av1"
	case "vp09":
		return "vp9"
	case "vp08":
		return "vp8"
	}
	return "unknown"
}

func probeWebM(r io.ReaderAt, size int64) (Info, error) {
	buf := make([]byte, min(size, 1<<16))
	n, _ := r.ReadAt(buf, 0)
	buf = buf[:n]
	info := Info{Container: "webm", Codec: "unknown"}
	for id, codec := range map[string]string{"V_AV1": "av1", "V_VP9": "vp9", "V_VP8": "vp8", "V_MPEG4/ISO/AVC": "h264", "V_MPEGH/ISO/HEVC": "hevc"} {
		if strings.Contains(string(buf), id) {
			info.Codec = codec
		}
	}
	return info, nil
}
