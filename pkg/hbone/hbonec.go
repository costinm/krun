package hbone

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"golang.org/x/net/http2"
)

// HboneCat copies stdin/stdout to a HBONE stream - without mTLS
// This is primarily used for testing and debug using SSH
func HboneCat(ug *http.Client, urlOrHost string, stdin io.ReadCloser, stdout io.WriteCloser) error {
	i, o := io.Pipe()

	fmt.Println("Connecting to ", urlOrHost)
	if strings.HasPrefix(urlOrHost, "http://") {
		// H2C - special case for debugging.
		ug = &http.Client{
			Transport: &http2.Transport{
				// So http2.Transport doesn't complain the URL scheme isn't 'https'
				AllowHTTP: true,
				// Pretend we are dialing a TLS endpoint.
				// Note, we ignore the passed tls.Config
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		}
	}


	if !strings.HasPrefix(urlOrHost, "https://") &&
			!strings.HasPrefix(urlOrHost, "http://") {
		h, p, err  := net.SplitHostPort(urlOrHost)
		if err != nil {
			return err
		}
		urlOrHost = "https://" + h + "/_hbone/" + p
	}


	r, err := http.NewRequest("POST", urlOrHost, i)
	if err != nil {
		return err
	}
	res, err := ug.Do(r)
	if err != nil {
		return err
	}

	go CopyBuffered(o, stdin)

	_, err = CopyBuffered(stdout, res.Body)
	return err
}

// HboneCatmTLS will proxy in/out (plain text) to a remote service, using mTLS tunnel over H2 POST.
func HboneCatmTLS(ug *http.Client, urlOrHost string, auth *Auth, stdin io.ReadCloser, stdout io.WriteCloser) error {
	i, o := io.Pipe()

	if !strings.HasPrefix(urlOrHost, "https://") {
		h, p, err  := net.SplitHostPort(urlOrHost)
		if err != nil {
			return err
		}
		urlOrHost = "https://" + h + "/_hbone/" + p
	}

	r, err := http.NewRequest("POST", urlOrHost, i)
	if err != nil {
		return err
	}
	res, err := ug.Do(r)
	if err != nil {
		return err
	}

	// TODO: Do the mTLS handshake

	go CopyBuffered(o, stdin)

	_, err = CopyBuffered(stdout, res.Body)
	return err
}
