package gnet

import (
	"crypto/tls"
	"fmt"
	exp "github.com/hsfish/guacamole_client_go"
	"github.com/hsfish/guacamole_client_go/gio"
	"net"
)

// SSLGuacamoleSocket ==> GuacamoleSocket
// * Provides abstract socket-like access to a Guacamole connection over a given
// * hostname and port.
type SSLGuacamoleSocket struct {
	reader gio.GuacamoleReader
	write  gio.GuacamoleWriter
	sock   *tls.Conn
}

// NewSSLGuacamoleSocket Construct & connect
//  * Creates a new InetGuacamoleSocket which reads and writes instructions
//  * to the Guacamole instruction stream of the Guacamole proxy server
//  * running at the given hostname and port.
//  *
//  * @param hostname The hostname of the Guacamole proxy server to connect to.
//  * @param port The port of the Guacamole proxy server to connect to.
//  * @throws GuacamoleException If an error occurs while connecting to the
//  *                            Guacamole proxy server.
func NewSSLGuacamoleSocket(hostname string, port int) (ret SSLGuacamoleSocket, err error) {
	// log.DebugF("Connecting to guacd at {}:{} via SSL/TLS.", hostname, port)

	// Get address
	address := fmt.Sprintf("%s:%d", hostname, port)

	// Connect with timeout
	sock, err := tls.DialWithDialer(
		&net.Dialer{Timeout: SocketTimeout}, "tcp", address,
		&tls.Config{
			InsecureSkipVerify: true,
		})
	if err != nil {
		// throw new GuacamoleUpstreamTimeoutException("Connection timed out.", e);
		return
	}

	// Set read timeout
	// On successful connect, retrieve I/O streams
	stream := gio.NewStream(sock, SocketTimeout)
	ret.sock = sock
	ret.reader = gio.NewReaderGuacamoleReader(stream)
	ret.write = gio.NewWriterGuacamoleWriter(stream)
	return
}

// Close Override GuacamoleSocket.Close
func (opt *SSLGuacamoleSocket) Close() (err exp.ExceptionInterface) {
	// logger.debug("Closing socket to guacd.");
	e := opt.sock.Close()
	if e != nil {
		err = exp.GuacamoleServerException.Throw(e.Error())
	}
	return
}

// GetReader Override GuacamoleSocket.GetReader
func (opt *SSLGuacamoleSocket) GetReader() (ret gio.GuacamoleReader) {
	ret = opt.reader
	return
}

// GetWriter Override GuacamoleSocket.GetWriter
func (opt *SSLGuacamoleSocket) GetWriter() (ret gio.GuacamoleWriter) {
	ret = opt.write
	return
}

// IsOpen Override GuacamoleSocket.IsOpen
func (opt *SSLGuacamoleSocket) IsOpen() (ok bool) {
	_, e := opt.sock.Write([]byte{})
	ok = e == nil
	return
}
