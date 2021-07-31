package hbone

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TODO: benchmark different sizes.
var bufSize = 32 * 1024

var (
	// createBuffer to get a buffer. io.Copy uses 32k.
	// experimental use shows ~20k max read with Firefox.
	bufferPoolCopy = sync.Pool{New: func() interface{} {
		return make([]byte, 0, 32*1024)
	}}
)

// CloseWriter is one of possible interfaces implemented by Out to send a FIN, without closing
// the input. Some writers only do this when Close is called.
type CloseWriter interface {
	CloseWrite() error
}

// CopyBuffered will copy src to dst, using a pooled intermediary buffer.
//
// Blocking, returns when src returned an error or EOF/graceful close.
// May also return with error if src or dst return errors.
func CopyBuffered(dst io.Writer, src io.Reader) (written int64, err error) {
	buf1 := bufferPoolCopy.Get().([]byte)
	defer bufferPoolCopy.Put(buf1)
	bufCap := cap(buf1)
	buf := buf1[0:bufCap:bufCap]

	// For netstack: src is a gonet.Conn, doesn't implement WriterTo. Dst is a net.TcpConn - and implements ReadFrom.
	// CopyBuffered is the actual implementation of Copy and CopyBuffer.
	// if buf is nil, one is allocated.
	// Duplicated from io

	// This will prevent stats from working.
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	//if wt, ok := src.(io.WriterTo); ok {
	//	return wt.WriteTo(dst)
	//}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	//if rt, ok := dst.(io.ReaderFrom); ok {
	//	return rt.ReadFrom(src)
	//}

	for {
		if srcc, ok := src.(net.Conn); ok {
			srcc.SetReadDeadline(time.Now().Add(15 * time.Minute))
		}
		nr, er := src.Read(buf)
		if er != nil && er != io.EOF {
			if strings.Contains(er.Error(), "NetworkIdleTimeout") {
				return written, io.EOF
			}
			return written, err
		}
		if nr == 0 {
			// shouldn't happen unless err == io.EOF
			return written, io.EOF
		}
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if f, ok := dst.(http.Flusher); ok {
				f.Flush()
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil { // == io.EOF
			return written, er
		}
	}
	return written, err
}


// HTTPConn wraps a http server request/response in a net.Conn
type HTTPConn struct {
	r            *http.Request
	w            http.ResponseWriter
	acceptedConn net.Conn
}

func (hc *HTTPConn) Read(b []byte) (n int, err error) {
	return hc.Read(b)
}

func (hc *HTTPConn) Write(b []byte) (n int, err error) {
	return hc.Write(b)
}

func (hc *HTTPConn) Close() error {
	// TODO: close write
	if cw, ok := hc.w.(CloseWriter); ok {
		return cw.CloseWrite()
	}
	log.Println("Unexpected writer not implement CloseWriter")
	return nil
}

func (hc *HTTPConn) LocalAddr() net.Addr {
	return hc.acceptedConn.LocalAddr()
}

func (hc *HTTPConn) RemoteAddr() net.Addr {
	return hc.acceptedConn.RemoteAddr()
}

func (hc *HTTPConn) SetDeadline(t time.Time) error {
	return nil
}

func (hc *HTTPConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (hc *HTTPConn) SetWriteDeadline(t time.Time) error {
	return nil
}

