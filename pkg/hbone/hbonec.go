package hbone

import (
	"io"
	"net"
	"net/http"
	"strings"
)

// HboneCat copies stdin/stdout to a HBONE stream - without mTLS
// This is primarily used for testing and debug using SSH
func HboneCat(ug *http.Client, urlOrHost string, tls string, stdin io.ReadCloser, stdout io.WriteCloser) error {
	i, o := io.Pipe()

	if !strings.HasPrefix(urlOrHost, "https://") {
		h, p, err  := net.SplitHostPort(urlOrHost)
		if err != nil {
			return err
		}
		urlOrHost = "https://" + h + "/_hbone/" + p
	}

	r, _ := http.NewRequest("POST", urlOrHost, i)
	res, err := ug.Transport.RoundTrip(r)
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

	r, _ := http.NewRequest("POST", urlOrHost, i)
	res, err := ug.Transport.RoundTrip(r)
	if err != nil {
		return err
	}

	// TODO: Do the mTLS handshake

	go CopyBuffered(o, stdin)

	_, err = CopyBuffered(stdout, res.Body)
	return err
}
