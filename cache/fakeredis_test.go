package cache

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRedis is a minimal in-process RESP2 server covering the commands the cache wrapper uses.
// It avoids a third-party test dependency (miniredis) and lets tests inject failures.
type fakeRedis struct {
	ln   net.Listener
	mu   sync.Mutex
	data map[string]string
	ttl  map[string]time.Duration
	fail atomic.Bool // answer every data command with an error
	hang atomic.Bool // never answer data commands (simulates a stuck server)
}

func newFakeRedis(t *testing.T) *fakeRedis {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	f := &fakeRedis{ln: ln, data: map[string]string{}, ttl: map[string]time.Duration{}}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(conn)
		}
	}()
	return f
}

func (f *fakeRedis) addr() string { return f.ln.Addr().String() }

func (f *fakeRedis) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}
		if _, err := io.WriteString(conn, f.handle(args)); err != nil {
			return
		}
	}
}

func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "*")))
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(hdr, "$")))
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}

func (f *fakeRedis) handle(args []string) string {
	cmd := strings.ToUpper(args[0])
	switch cmd {
	case "HELLO":
		return "-ERR unknown command 'HELLO'\r\n"
	case "CLIENT", "SELECT", "AUTH":
		return "+OK\r\n"
	}
	if f.hang.Load() {
		time.Sleep(2 * time.Second)
		return "+OK\r\n"
	}
	if f.fail.Load() {
		return "-ERR boom\r\n"
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	switch cmd {
	case "PING":
		return "+PONG\r\n"
	case "SET":
		f.data[args[1]] = args[2]
		for i := 3; i+1 < len(args); i++ {
			switch strings.ToUpper(args[i]) {
			case "EX":
				s, _ := strconv.Atoi(args[i+1])
				f.ttl[args[1]] = time.Duration(s) * time.Second
			case "PX":
				ms, _ := strconv.Atoi(args[i+1])
				f.ttl[args[1]] = time.Duration(ms) * time.Millisecond
			}
		}
		return "+OK\r\n"
	case "GET":
		v, ok := f.data[args[1]]
		if !ok {
			return "$-1\r\n"
		}
		return fmt.Sprintf("$%d\r\n%s\r\n", len(v), v)
	case "DEL":
		n := 0
		for _, k := range args[1:] {
			if _, ok := f.data[k]; ok {
				delete(f.data, k)
				delete(f.ttl, k)
				n++
			}
		}
		return fmt.Sprintf(":%d\r\n", n)
	case "EXISTS":
		n := 0
		for _, k := range args[1:] {
			if _, ok := f.data[k]; ok {
				n++
			}
		}
		return fmt.Sprintf(":%d\r\n", n)
	case "EXPIRE", "PEXPIRE":
		if _, ok := f.data[args[1]]; !ok {
			return ":0\r\n"
		}
		n, _ := strconv.Atoi(args[2])
		unit := time.Second
		if cmd == "PEXPIRE" {
			unit = time.Millisecond
		}
		f.ttl[args[1]] = time.Duration(n) * unit
		return ":1\r\n"
	}
	return "-ERR unknown command\r\n"
}

func (f *fakeRedis) ttlOf(key string) time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ttl[key]
}
