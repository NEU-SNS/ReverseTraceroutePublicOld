/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package yaml

// Set the writer error and return false.
func yaml_emitter_set_writer_error(emitter *yaml_emitter_t, problem string) bool {
	emitter.error = yaml_WRITER_ERROR
	emitter.problem = problem
	return false
}

// Flush the output buffer.
func yaml_emitter_flush(emitter *yaml_emitter_t) bool {
	if emitter.write_handler == nil {
		panic("write handler not set")
	}

	// Check if the buffer is empty.
	if emitter.buffer_pos == 0 {
		return true
	}

	// If the output encoding is UTF-8, we don't need to recode the buffer.
	if emitter.encoding == yaml_UTF8_ENCODING {
		if err := emitter.write_handler(emitter, emitter.buffer[:emitter.buffer_pos]); err != nil {
			return yaml_emitter_set_writer_error(emitter, "write error: "+err.Error())
		}
		emitter.buffer_pos = 0
		return true
	}

	// Recode the buffer into the raw buffer.
	var low, high int
	if emitter.encoding == yaml_UTF16LE_ENCODING {
		low, high = 0, 1
	} else {
		high, low = 1, 0
	}

	pos := 0
	for pos < emitter.buffer_pos {
		// See the "reader.c" code for more details on UTF-8 encoding.  Note
		// that we assume that the buffer contains a valid UTF-8 sequence.

		// Read the next UTF-8 character.
		octet := emitter.buffer[pos]

		var w int
		var value rune
		switch {
		case octet&0x80 == 0x00:
			w, value = 1, rune(octet&0x7F)
		case octet&0xE0 == 0xC0:
			w, value = 2, rune(octet&0x1F)
		case octet&0xF0 == 0xE0:
			w, value = 3, rune(octet&0x0F)
		case octet&0xF8 == 0xF0:
			w, value = 4, rune(octet&0x07)
		}
		for k := 1; k < w; k++ {
			octet = emitter.buffer[pos+k]
			value = (value << 6) + (rune(octet) & 0x3F)
		}
		pos += w

		// Write the character.
		if value < 0x10000 {
			var b [2]byte
			b[high] = byte(value >> 8)
			b[low] = byte(value & 0xFF)
			emitter.raw_buffer = append(emitter.raw_buffer, b[0], b[1])
		} else {
			// Write the character using a surrogate pair (check "reader.c").
			var b [4]byte
			value -= 0x10000
			b[high] = byte(0xD8 + (value >> 18))
			b[low] = byte((value >> 10) & 0xFF)
			b[high+2] = byte(0xDC + ((value >> 8) & 0xFF))
			b[low+2] = byte(value & 0xFF)
			emitter.raw_buffer = append(emitter.raw_buffer, b[0], b[1], b[2], b[3])
		}
	}

	// Write the raw buffer.
	if err := emitter.write_handler(emitter, emitter.raw_buffer); err != nil {
		return yaml_emitter_set_writer_error(emitter, "write error: "+err.Error())
	}
	emitter.buffer_pos = 0
	emitter.raw_buffer = emitter.raw_buffer[:0]
	return true
}
