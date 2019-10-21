package gservlet

import (
	"fmt"
	"strings"

	exp "github.com/hsfish/guacamole_client_go"
	"github.com/hsfish/guacamole_client_go/gio"
	"github.com/hsfish/guacamole_client_go/gnet"

	logger "github.com/Sirupsen/logrus"
)

/**
 * Logger for this class.
 */
//private final Logger logger = LoggerFactory.getLogger(GuacamoleHTTPTunnelServlet.class);

const (
	/*READ_PREFIX *
	 * The prefix of the query string which denotes a tunnel read operation.
	 */
	READ_PREFIX string = "read:"

	/*WRITE_PREFIX *
	 * The prefix of the query string which denotes a tunnel write operation.
	 */
	WRITE_PREFIX string = "write:"

	/*READ_PREFIX_LENGTH *
	 * The length of the read prefix, in characters.
	 */
	READ_PREFIX_LENGTH = len(READ_PREFIX)

	/*WRITE_PREFIX_LENGTH *
	 * The length of the write prefix, in characters.
	 */
	WRITE_PREFIX_LENGTH = len(WRITE_PREFIX)

	/*UUID_LENGTH *
	 * The length of every tunnel UUID, in characters.
	 */
	UUID_LENGTH = 36
)

/*GuacamoleHTTPTunnelServlet ==> HttpServlet*
 * A HttpServlet implementing and abstracting the operations required by the
 * HTTP implementation of the JavaScript Guacamole client's tunnel.
 */
type GuacamoleHTTPTunnelServlet struct {
	/**
	 * Map of absolutely all active tunnels using HTTP, indexed by tunnel UUID.
	 */
	tunnels GuacamoleHTTPTunnelMap

	/**
	 * Called whenever the JavaScript Guacamole client makes a connection
	 * request via HTTP. It it up to the implementor of this function to define
	 * what conditions must be met for a tunnel to be configured and returned
	 * as a result of this connection request (whether some sort of credentials
	 * must be specified, for example).
	 *
	 * @param request
	 *     The HttpServletRequest associated with the connection request
	 *     received. Any parameters specified along with the connection request
	 *     can be read from this object.
	 *
	 * @return
	 *     A newly constructed GuacamoleTunnel if successful, null otherwise.
	 *
	 * @throws GuacamoleException
	 *     If an error occurs while constructing the GuacamoleTunnel, or if the
	 *     conditions required for connection are not met.
	 */
	doConnect     DoConnectInterface
	doSuccConnect DoConnectSuccInterface
	// doStopConnect DoConnectStopInterface
}

// NewGuacamoleHTTPTunnelServlet Construct funtion
func NewGuacamoleHTTPTunnelServlet(doConnect DoConnectInterface, doSuccConnect DoConnectSuccInterface, doStopConnect DoConnectStopInterface) (ret GuacamoleHTTPTunnelServlet) {
	ret.tunnels = NewGuacamoleHTTPTunnelMap(doStopConnect)
	ret.doConnect = doConnect
	ret.doSuccConnect = doSuccConnect
	// ret.doStopConnect = doStopConnect
	return
}

/**
 * Registers the given tunnel such that future read/write requests to that
 * tunnel will be properly directed.
 *
 * @param tunnel
 *     The tunnel to register.
 */
func (opt *GuacamoleHTTPTunnelServlet) registerTunnel(tunnel gnet.GuacamoleTunnel) {
	opt.tunnels.Put(tunnel.GetUUID().String(), tunnel)
	logger.Debugf("Registered tunnel \"%v\".", tunnel.GetUUID())
	opt.doSuccConnect(tunnel)
}

/**
 * Deregisters the given tunnel such that future read/write requests to
 * that tunnel will be rejected.
 *
 * @param tunnel
 *     The tunnel to deregister.
 */
func (opt *GuacamoleHTTPTunnelServlet) deregisterTunnel(tunnel gnet.GuacamoleTunnel) {
	opt.tunnels.Remove(tunnel.GetUUID().String())
	logger.Debugf("Deregistered tunnel \"%v\".", tunnel.GetUUID())
	// opt.doStopConnect(tunnel)
}

/**
 * Returns the tunnel with the given UUID, if it has been registered with
 * registerTunnel() and not yet deregistered with deregisterTunnel().
 *
 * @param tunnelUUID
 *     The UUID of registered tunnel.
 *
 * @return
 *     The tunnel corresponding to the given UUID.
 *
 * @throws GuacamoleException
 *     If the requested tunnel does not exist because it has not yet been
 *     registered or it has been deregistered.
 */
func (opt *GuacamoleHTTPTunnelServlet) getTunnel(tunnelUUID string) (ret gnet.GuacamoleTunnel,
	err exp.ExceptionInterface) {

	// Pull tunnel from map
	ret, ok := opt.tunnels.Get(tunnelUUID)

	if !ok {
		err = exp.GuacamoleResourceNotFoundException.Throw("No such tunnel.")
	}
	return
}

//DoGet @Override
func (opt *GuacamoleHTTPTunnelServlet) DoGet(request HTTPServletRequestInterface, response HTTPServletResponseInterface) error {
	return opt.HandleTunnelRequest(request, response)
}

//DoPost @Override
func (opt *GuacamoleHTTPTunnelServlet) DoPost(request HTTPServletRequestInterface, response HTTPServletResponseInterface) error {
	return opt.HandleTunnelRequest(request, response)
}

/**
 * Sends an error on the given HTTP response using the information within
 * the given GuacamoleStatus.
 *
 * @param response
 *     The HTTP response to use to send the error.
 *
 * @param guacStatus
 *     The status to send
 *
 * @param message
 *     A human-readable message that can be presented to the user.
 *
 * @throws ServletException
 *     If an error prevents sending of the error code.
 */
func (opt *GuacamoleHTTPTunnelServlet) sendError(response HTTPServletResponseInterface,
	guacStatus exp.GuacamoleStatus, message string) (err error) {

	committed, err := response.IsCommitted()
	if err != nil {
		// If unable to send error at all due to I/O problems,
		// rethrow as servlet exception
		return
	}

	// If response not committed, send error code and message
	if !committed {
		response.AddHeader("Guacamole-Status-Code", fmt.Sprintf("%v", guacStatus.GetGuacamoleStatusCode()))
		response.AddHeader("Guacamole-Error-Message", message)
		err = response.SendError(guacStatus.GetHTTPStatusCode())
	}
	return
}

/*HandleTunnelRequest put it into GET/POST handle *
 * Dispatches every HTTP GET and POST request to the appropriate handler
 * function based on the query string.
 *
 * @param request
 *     The HttpServletRequest associated with the GET or POST request
 *     received.
 *
 * @param response
 *     The HttpServletResponse associated with the GET or POST request
 *     received.
 *
 * @throws ServletException
 *     If an error occurs while servicing the request.
 */
func (opt *GuacamoleHTTPTunnelServlet) HandleTunnelRequest(request HTTPServletRequestInterface,
	response HTTPServletResponseInterface) (e error) {

	err := opt.handleTunnelRequestCore(request, response)
	if err == nil {
		return
	}
	switch err.Kind() {
	case exp.GuacamoleClientException:
		logger.Warn("HTTP tunnel request rejected: ", err.GetMessage())
		e = opt.sendError(response, err.GetStatus(), err.GetMessage())
	default:
		logger.Error("HTTP tunnel request failed: ", err.GetMessage())
		logger.Debug("Internal error in HTTP tunnel.", err)
		e = opt.sendError(response, err.GetStatus(), "Internal server error.")
	}
	return
}

func (opt *GuacamoleHTTPTunnelServlet) handleTunnelRequestCore(request HTTPServletRequestInterface,
	response HTTPServletResponseInterface) (err exp.ExceptionInterface) {
	query := request.GetQueryString()
	if len(query) == 0 {
		return exp.GuacamoleClientException.Throw("No query string provided.")
	}
	// If connect operation, call doConnect() and return tunnel UUID
	// in response.
	if query == "connect" {

		tunnel, e := opt.doConnect(request)
		// Failed to connect
		if tunnel == nil || e != nil {
			switch e.(type) {
			case exp.ExceptionInterface:
				err = e.(exp.ExceptionInterface)
			default:
				err = exp.GuacamoleResourceNotFoundException.Throw("No tunnel created.", e.Error())
			}
			return
		}
		// Register newly-created tunnel
		opt.registerTunnel(tunnel)

		// Ensure buggy browsers do not cache response
		response.SetHeader("Cache-Control", "no-cache")

		// Send UUID to client
		e = response.WriteString(tunnel.GetUUID().String())

		if e != nil {
			err = exp.GuacamoleServerException.Throw(e.Error())
			return
		}

	} else if strings.HasPrefix(query, READ_PREFIX) {
		// If read operation, call doRead() with tunnel UUID, ignoring any
		// characters following the tunnel UUID.
		err = opt.doRead(request, response, query[READ_PREFIX_LENGTH:READ_PREFIX_LENGTH+UUID_LENGTH])
	} else if strings.HasPrefix(query, (WRITE_PREFIX)) {
		// If write operation, call doWrite() with tunnel UUID, ignoring any
		// characters following the tunnel UUID.
		err = opt.doWrite(request, response, query[WRITE_PREFIX_LENGTH:WRITE_PREFIX_LENGTH+UUID_LENGTH])
	} else {
		// Otherwise, invalid operation
		err = exp.GuacamoleClientException.Throw("Invalid tunnel operation: " + query)
	}

	return
}

/**
 * Called whenever the JavaScript Guacamole client makes a read request.
 * This function should in general not be overridden, as it already
 * contains a proper implementation of the read operation.
 *
 * @param request
 *     The HttpServletRequest associated with the read request received.
 *
 * @param response
 *     The HttpServletResponse associated with the write request received.
 *     Any data to be sent to the client in response to the write request
 *     should be written to the response body of this HttpServletResponse.
 *
 * @param tunnelUUID
 *     The UUID of the tunnel to read from, as specified in the write
 *     request. This tunnel must have been created by a previous call to
 *     doConnect().
 *
 * @throws GuacamoleException
 *     If an error occurs while handling the read request.
 */
func (opt *GuacamoleHTTPTunnelServlet) doRead(request HTTPServletRequestInterface,
	response HTTPServletResponseInterface, tunnelUUID string) (err exp.ExceptionInterface) {

	// Get tunnel, ensure tunnel exists
	tunnel, err := opt.getTunnel(tunnelUUID)
	if err != nil {
		return
	}

	// Ensure tunnel is open
	if !tunnel.IsOpen() {
		opt.deregisterTunnel(tunnel)
		err = exp.GuacamoleResourceNotFoundException.Throw("Tunnel is closed.")
		return
	}

	reader := tunnel.AcquireReader()
	defer tunnel.ReleaseReader()

	e := opt.doReadCore1(response, reader, tunnel)

	if e != nil {

		// Log typically frequent I/O error if desired
		logger.Debug("Error writing to servlet output stream", e)

		// Deregister and close
		opt.deregisterTunnel(tunnel)
		tunnel.Close()
	}

	return
}

func (opt *GuacamoleHTTPTunnelServlet) doReadCore1(response HTTPServletResponseInterface,
	reader gio.GuacamoleReader, tunnel gnet.GuacamoleTunnel) (e exp.ExceptionInterface) {
	// Note that although we are sending text, Webkit browsers will
	// buffer 1024 bytes before starting a normal stream if we use
	// anything but application/octet-stream.
	response.SetContentType("application/octet-stream")
	response.SetHeader("Cache-Control", "no-cache")

	// response.Close() -->
	defer response.FlushBuffer()

	// Get writer for response
	// Writer out = new BufferedWriter(new OutputStreamWriter(response.getOutputStream(), "UTF-8"));

	// Stream data to response, ensuring output stream is closed
	err := opt.doReadCore2(response, reader, tunnel)

	if err == nil {
		// success
		return
	}

	switch err.Kind() {
	// Send end-of-stream marker and close tunnel if connection is closed
	case exp.GuacamoleConnectionClosedException:
		// Deregister and close
		opt.deregisterTunnel(tunnel)
		tunnel.Close()

		// End-of-instructions marker
		response.WriteString("0.;")
		response.FlushBuffer()
	default:
		// Deregister and close
		e = err
	}
	return
}

func (opt *GuacamoleHTTPTunnelServlet) doReadCore2(response HTTPServletResponseInterface,
	reader gio.GuacamoleReader, tunnel gnet.GuacamoleTunnel) (err exp.ExceptionInterface) {
	var ok bool
	var message []byte
	// Deregister tunnel and throw error if we reach EOF without
	// having ever sent any data
	message, err = reader.Read()
	if err != nil {
		return
	}
	// start := time.Now().UnixNano()
	// For all messages, until another stream is ready (we send at least one message)
	for ; tunnel.IsOpen() && len(message) > 0 && err == nil; message, err = reader.Read() {
		// time.Sleep(2000)
		// Get message output bytes
		e := response.Write(message)
		if e != nil {
			err = exp.GuacamoleException.Throw(e.Error())
			return
		}

		// Flush if we expect to wait
		ok, err = reader.Available()
		if err != nil {
			return
		}

		if !ok {
			e = response.FlushBuffer()
			if e != nil {
				err = exp.GuacamoleException.Throw(e.Error())
				return
			}
		}
		if strings.Contains(string(message), "error") {
			response.Write([]byte("10.disconnect;"))
			response.FlushBuffer()
			tunnel.GetConfiguration().SetParameter("errMsg", string(message))
			opt.deregisterTunnel(tunnel)
			tunnel.Close()
		}
		// No more messages another stream can take over
		if tunnel.HasQueuedReaderThreads() {
			break
		}
		// if time.Now().UnixNano()-start > 1000000 {
		// 	break
		// }
	}
	// Close tunnel immediately upon EOF
	if err != nil {
		opt.deregisterTunnel(tunnel)
		tunnel.Close()
	}

	// End-of-instructions marker
	response.WriteString("0.;")
	response.FlushBuffer()
	return nil
}

/**
 * Called whenever the JavaScript Guacamole client makes a write request.
 * This function should in general not be overridden, as it already
 * contains a proper implementation of the write operation.
 *
 * @param request
 *     The HttpServletRequest associated with the write request received.
 *     Any data to be written will be specified within the body of this
 *     request.
 *
 * @param response
 *     The HttpServletResponse associated with the write request received.
 *
 * @param tunnelUUID
 *     The UUID of the tunnel to write to, as specified in the write
 *     request. This tunnel must have been created by a previous call to
 *     doConnect().
 *
 * @throws GuacamoleException
 *     If an error occurs while handling the write request.
 */
func (opt *GuacamoleHTTPTunnelServlet) doWrite(request HTTPServletRequestInterface,
	response HTTPServletResponseInterface, tunnelUUID string) (err exp.ExceptionInterface) {
	tunnel, err := opt.getTunnel(tunnelUUID)
	if err != nil {
		return
	}

	// We still need to set the content type to avoid the default of
	// text/html, as such a content type would cause some browsers to
	// attempt to parse the result, even though the JavaScript client
	// does not explicitly request such parsing.
	response.SetContentType("application/octet-stream")
	response.SetHeader("Cache-Control", "no-cache")
	response.SetContentLength(0)

	writer := tunnel.AcquireWriter()
	defer tunnel.ReleaseWriter()

	var e error
	length := 0
	buffer := make([]byte, 8192, 8192)
	for length, e = request.Read(buffer); tunnel.IsOpen() && length > 0; length, e = request.Read(buffer) {
		err = writer.Write(buffer, 0, length)
		if err != nil {
			break
		}
	}
	if e != nil {
		// EOF
		// No need to close Request stream
		return nil
	}

	if err != nil {
		opt.deregisterTunnel(tunnel)
		tunnel.Close()
	}
	return
}

// Destroy release
func (opt *GuacamoleHTTPTunnelServlet) Destroy() {
	opt.tunnels.Shutdown()
}

func (opt *GuacamoleHTTPTunnelServlet) GetTunnel(tunnelUUID string) (ret gnet.GuacamoleTunnel, err error) {

	// Pull tunnel from map
	ret, ok := opt.tunnels.Get(tunnelUUID)

	if !ok {
		err = fmt.Errorf("No such tunnel")
	}
	return
}
