package scanning

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClamAVScannerVerdicts(t *testing.T) {
	for _, tc := range []struct {
		name, response, want string
	}{
		{"clean", "stream: OK\x00", StatusClean},
		{"infected", "stream: Synthetic-Test FOUND\x00", StatusInfected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, server := net.Pipe()
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer server.Close()
				command := make([]byte, len("zINSTREAM\x00"))
				_, _ = io.ReadFull(server, command)
				for {
					var size [4]byte
					if _, readErr := io.ReadFull(server, size[:]); readErr != nil {
						return
					}
					length := binary.BigEndian.Uint32(size[:])
					if length == 0 {
						break
					}
					_, _ = io.CopyN(io.Discard, server, int64(length))
				}
				_, _ = io.WriteString(server, tc.response)
			}()
			scanner := ClamAVScanner{Timeout: time.Second, dial: func(context.Context) (net.Conn, error) { return client, nil }}
			status, err := scanner.Scan(t.Context(), []byte("synthetic"))
			require.NoError(t, err)
			require.Equal(t, tc.want, status)
			<-done
		})
	}
}
