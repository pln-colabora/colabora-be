package scanning

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"time"
)

type ClamAVScanner struct {
	Address string
	Timeout time.Duration
	dial    func(context.Context) (net.Conn, error)
}

func (s ClamAVScanner) Scan(ctx context.Context, content []byte) (string, error) {
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	var conn net.Conn
	var err error
	if s.dial != nil {
		conn, err = s.dial(ctx)
	} else {
		conn, err = (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", s.Address)
	}
	if err != nil {
		return "", errors.New("scanner unavailable")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if err = writeAll(conn, []byte("zINSTREAM\x00")); err != nil {
		return "", errors.New("scanner write failed")
	}
	for offset := 0; offset < len(content); {
		end := offset + 32*1024
		if end > len(content) {
			end = len(content)
		}
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(end-offset))
		if err = writeAll(conn, size[:]); err != nil {
			return "", errors.New("scanner write failed")
		}
		if err = writeAll(conn, content[offset:end]); err != nil {
			return "", errors.New("scanner write failed")
		}
		offset = end
	}
	if err = writeAll(conn, []byte{0, 0, 0, 0}); err != nil {
		return "", errors.New("scanner write failed")
	}
	response, err := bufio.NewReader(io.LimitReader(conn, 4097)).ReadString(0)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", errors.New("scanner response failed")
	}
	if len(response) > 4096 {
		return "", errors.New("scanner response too large")
	}
	response = strings.TrimSpace(strings.TrimSuffix(response, "\x00"))
	switch {
	case strings.HasSuffix(response, " OK"):
		return StatusClean, nil
	case strings.HasSuffix(response, " FOUND"):
		return StatusInfected, nil
	default:
		return "", errors.New("scanner returned an unknown result")
	}
}

func writeAll(writer io.Writer, content []byte) error {
	for len(content) > 0 {
		written, err := writer.Write(content)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		content = content[written:]
	}
	return nil
}
