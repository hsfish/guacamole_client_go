package gio

// Move FilteredGuacamoleWriter from protocol folder to here
// Avoid cross depends

import (
	exp "github.com/hsfish/guacamole_client_go"
	"github.com/hsfish/guacamole_client_go/gprotocol"
)

// FilteredGuacamoleWriter ==> GuacamoleWriter
//  * GuacamoleWriter which applies a given GuacamoleFilter to observe or alter
//  * all written instructions. Instructions may also be dropped or denied by
//  * the filter.
type FilteredGuacamoleWriter struct {
	/**
	 * The wrapped GuacamoleWriter.
	 */
	writer GuacamoleWriter

	/**
	 * The filter to apply when writing instructions.
	 */
	filter gprotocol.GuacamoleFilter

	/**
	 * Parser for reading instructions prior to writing, such that they can be
	 * passed on to the filter.
	 */
	parser gprotocol.GuacamoleParser
}

// NewFilteredGuacamoleWriter *
// * Wraps the given GuacamoleWriter, applying the given filter to all written
// * instructions. Future writes will only write instructions which pass
// * the filter.
// *
// * @param writer The GuacamoleWriter to wrap.
// * @param filter The filter which dictates which instructions are written,
// *               and how.
func NewFilteredGuacamoleWriter(writer GuacamoleWriter, filter gprotocol.GuacamoleFilter) (ret FilteredGuacamoleWriter) {
	ret.writer = writer
	ret.filter = filter
	ret.parser = gprotocol.NewGuacamoleParser()
	return
}

// Write override GuacamoleWriter.Write
func (opt *FilteredGuacamoleWriter) Write(chunk []byte, offset, length int) (err exp.ExceptionInterface) {
	for length > 0 {
		var parsed int
		for parsed, err = opt.parser.Append(chunk, offset, length); parsed > 0 && err == nil; parsed, err = opt.parser.Append(chunk, offset, length) {
			offset += parsed
			length -= parsed
		}
		if err != nil {
			return
		}
		if !opt.parser.HasNext() {
			err = exp.GuacamoleServerException.Throw("Filtered write() contained an incomplete instruction.")
			return
		}

		if v, ok := opt.parser.Next(); ok {
			err = opt.WriteInstruction(v)
			if err != nil {
				return
			}
		}
	}
	return
}

// WriteAll override GuacamoleWriter.WriteAll
func (opt *FilteredGuacamoleWriter) WriteAll(chunk []byte) (err exp.ExceptionInterface) {
	return opt.Write(chunk, 0, len(chunk))
}

// WriteInstruction override GuacamoleWriter.WriteInstruction
func (opt *FilteredGuacamoleWriter) WriteInstruction(instruction gprotocol.GuacamoleInstruction) (err exp.ExceptionInterface) {

	// Write instruction only if not dropped
	filteredInstruction, err := opt.filter.Filter(instruction)
	if err != nil {
		return
	}
	if len(filteredInstruction.GetOpcode()) > 0 {
		err = opt.writer.WriteInstruction(filteredInstruction)
	}
	return
}
