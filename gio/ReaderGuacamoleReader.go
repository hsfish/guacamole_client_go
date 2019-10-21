package gio

import (
	"net"
	"strconv"

	exp "github.com/hsfish/guacamole_client_go"
	"github.com/hsfish/guacamole_client_go/gprotocol"
)

// ReaderGuacamoleReader A GuacamoleReader which wraps a standard io Reader,
// using that Reader as the Guacamole instruction stream.
type ReaderGuacamoleReader struct {
	input      *Stream
	parseStart int
	buffer     []rune
	usedLength int
}

// NewReaderGuacamoleReader Construct function of ReaderGuacamoleReader
func NewReaderGuacamoleReader(input *Stream) (ret GuacamoleReader) {
	one := ReaderGuacamoleReader{}
	one.input = input
	one.parseStart = 0
	one.buffer = make([]rune, 0, 20480)
	ret = &one
	return
}

// Available override GuacamoleReader.Available
func (opt *ReaderGuacamoleReader) Available() (ok bool, err exp.ExceptionInterface) {
	ok = len(opt.buffer) > 0
	if ok {
		return
	}
	ok, e := opt.input.Available()
	if e != nil {
		err = exp.GuacamoleServerException.Throw(e.Error())
		return
	}
	return
}

// Read override GuacamoleReader.Read
func (opt *ReaderGuacamoleReader) Read() (instruction []byte, err exp.ExceptionInterface) {

mainLoop:
	// While we're blocking, or input is available
	for {
		// Length of element
		var elementLength int

		// Resume where we left off
		i := opt.parseStart

	parseLoop:
		// Parse instruction in buffer
		for i < len(opt.buffer) {
			// Read character
			readChar := opt.buffer[i]
			i++

			switch readChar {
			// If digit, update length
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				elementLength = elementLength*10 + int(readChar-'0')

			// If not digit, check for end-of-length character
			case '.':
				if i+elementLength >= len(opt.buffer) {
					// break for i < opt.usedLength { ... }
					// Otherwise, read more data
					break parseLoop
				}
				// Check if element present in buffer
				terminator := opt.buffer[i+elementLength]
				// Move to character after terminator
				i += elementLength + 1

				// Reset length
				elementLength = 0

				// Continue here if necessary
				opt.parseStart = i

				// If terminator is semicolon, we have a full
				// instruction.
				switch terminator {
				case ';':
					instruction = []byte(string(opt.buffer[0:i]))
					opt.parseStart = 0
					opt.buffer = opt.buffer[i:]
					break mainLoop
				case ',':
					// nothing
				default:
					err = exp.GuacamoleServerException.Throw("Element terminator of instruction was not ';' nor ','")
					break mainLoop
				}
			default:
				// Otherwise, parse error
				err = exp.GuacamoleServerException.Throw("Non-numeric character in element length.")
				break mainLoop
			}

		}

		// no more buffer explain in golang
		// discard
		// if (usedLength > buffer.length/2) { ... }
		// using
		// Read(stepBuffer)
		// buffer = buffer + stepBuffer[0:n]
		// instead

		stepBuffer, e := opt.input.Read()
		if e != nil {
			// Discard
			// Time out throw GuacamoleUpstreamTimeoutException for
			// Closed throw GuacamoleConnectionClosedException for
			// Other socket err
			// Here or use normal err instead

			// Inside opt.input.Read()
			// Error occurs will close socket
			// So ...
			switch e.(type) {
			case net.Error:
				ex := e.(net.Error)
				if ex.Timeout() {
					err = exp.GuacamoleUpstreamTimeoutException.Throw("Connection to guacd timed out.", e.Error())
				} else {
					err = exp.GuacamoleConnectionClosedException.Throw("Connection to guacd is closed.", e.Error())
				}
			default:
				err = exp.GuacamoleServerException.Throw(e.Error())
			}
			break mainLoop
		}
		opt.buffer = append(opt.buffer, []rune(string(stepBuffer))...)
	}
	return
}

// ReadInstruction override GuacamoleReader.ReadInstruction
func (opt *ReaderGuacamoleReader) ReadInstruction() (instruction gprotocol.GuacamoleInstruction, err exp.ExceptionInterface) {

	// Get instruction
	instructionBuffer, err := opt.Read()

	// If EOF, return EOF
	if err != nil {
		return
	}

	// Start of element
	elementStart := 0

	// Build list of elements
	elements := make([]string, 0, 1)
	for elementStart < len(instructionBuffer) {
		// Find end of length
		lengthEnd := -1
		for i := elementStart; i < len(instructionBuffer); i++ {
			if instructionBuffer[i] == '.' {
				lengthEnd = i
				break
			}
		}
		// read() is required to return a complete instruction. If it does
		// not, this is a severe internal error.
		if lengthEnd == -1 {
			err = exp.GuacamoleServerException.Throw("Read returned incomplete instruction.")
			return
		}

		// Parse length
		length, e := strconv.Atoi(string(instructionBuffer[elementStart:lengthEnd]))
		if e != nil {
			err = exp.GuacamoleServerException.Throw("Read returned wrong pattern instruction.", e.Error())
			return
		}

		// Parse element from just after period
		elementStart = lengthEnd + 1
		element := string(instructionBuffer[elementStart : elementStart+length])

		// Append element to list of elements
		elements = append(elements, element)

		// Read terminator after element
		elementStart += length
		terminator := instructionBuffer[elementStart]

		// Continue reading instructions after terminator
		elementStart++

		// If we've reached the end of the instruction
		if terminator == ';' {
			break
		}

	}

	// Pull opcode off elements list
	// Create instruction
	instruction = gprotocol.NewGuacamoleInstruction(elements[0], elements[1:]...)

	// Return parsed instruction
	return
}
