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

import (
	"bytes"
)

// The parser implements the following grammar:
//
// stream               ::= STREAM-START implicit_document? explicit_document* STREAM-END
// implicit_document    ::= block_node DOCUMENT-END*
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
// block_node_or_indentless_sequence    ::=
//                          ALIAS
//                          | properties (block_content | indentless_block_sequence)?
//                          | block_content
//                          | indentless_block_sequence
// block_node           ::= ALIAS
//                          | properties block_content?
//                          | block_content
// flow_node            ::= ALIAS
//                          | properties flow_content?
//                          | flow_content
// properties           ::= TAG ANCHOR? | ANCHOR TAG?
// block_content        ::= block_collection | flow_collection | SCALAR
// flow_content         ::= flow_collection | SCALAR
// block_collection     ::= block_sequence | block_mapping
// flow_collection      ::= flow_sequence | flow_mapping
// block_sequence       ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
// indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
// block_mapping        ::= BLOCK-MAPPING_START
//                          ((KEY block_node_or_indentless_sequence?)?
//                          (VALUE block_node_or_indentless_sequence?)?)*
//                          BLOCK-END
// flow_sequence        ::= FLOW-SEQUENCE-START
//                          (flow_sequence_entry FLOW-ENTRY)*
//                          flow_sequence_entry?
//                          FLOW-SEQUENCE-END
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
// flow_mapping         ::= FLOW-MAPPING-START
//                          (flow_mapping_entry FLOW-ENTRY)*
//                          flow_mapping_entry?
//                          FLOW-MAPPING-END
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?

// Peek the next token in the token queue.
func peek_token(parser *yaml_parser_t) *yaml_token_t {
	if parser.token_available || yaml_parser_fetch_more_tokens(parser) {
		return &parser.tokens[parser.tokens_head]
	}
	return nil
}

// Remove the next token from the queue (must be called after peek_token).
func skip_token(parser *yaml_parser_t) {
	parser.token_available = false
	parser.tokens_parsed++
	parser.stream_end_produced = parser.tokens[parser.tokens_head].typ == yaml_STREAM_END_TOKEN
	parser.tokens_head++
}

// Get the next event.
func yaml_parser_parse(parser *yaml_parser_t, event *yaml_event_t) bool {
	// Erase the event object.
	*event = yaml_event_t{}

	// No events after the end of the stream or error.
	if parser.stream_end_produced || parser.error != yaml_NO_ERROR || parser.state == yaml_PARSE_END_STATE {
		return true
	}

	// Generate the next event.
	return yaml_parser_state_machine(parser, event)
}

// Set parser error.
func yaml_parser_set_parser_error(parser *yaml_parser_t, problem string, problem_mark yaml_mark_t) bool {
	parser.error = yaml_PARSER_ERROR
	parser.problem = problem
	parser.problem_mark = problem_mark
	return false
}

func yaml_parser_set_parser_error_context(parser *yaml_parser_t, context string, context_mark yaml_mark_t, problem string, problem_mark yaml_mark_t) bool {
	parser.error = yaml_PARSER_ERROR
	parser.context = context
	parser.context_mark = context_mark
	parser.problem = problem
	parser.problem_mark = problem_mark
	return false
}

// State dispatcher.
func yaml_parser_state_machine(parser *yaml_parser_t, event *yaml_event_t) bool {
	//trace("yaml_parser_state_machine", "state:", parser.state.String())

	switch parser.state {
	case yaml_PARSE_STREAM_START_STATE:
		return yaml_parser_parse_stream_start(parser, event)

	case yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE:
		return yaml_parser_parse_document_start(parser, event, true)

	case yaml_PARSE_DOCUMENT_START_STATE:
		return yaml_parser_parse_document_start(parser, event, false)

	case yaml_PARSE_DOCUMENT_CONTENT_STATE:
		return yaml_parser_parse_document_content(parser, event)

	case yaml_PARSE_DOCUMENT_END_STATE:
		return yaml_parser_parse_document_end(parser, event)

	case yaml_PARSE_BLOCK_NODE_STATE:
		return yaml_parser_parse_node(parser, event, true, false)

	case yaml_PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE:
		return yaml_parser_parse_node(parser, event, true, true)

	case yaml_PARSE_FLOW_NODE_STATE:
		return yaml_parser_parse_node(parser, event, false, false)

	case yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE:
		return yaml_parser_parse_block_sequence_entry(parser, event, true)

	case yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_block_sequence_entry(parser, event, false)

	case yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_indentless_sequence_entry(parser, event)

	case yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE:
		return yaml_parser_parse_block_mapping_key(parser, event, true)

	case yaml_PARSE_BLOCK_MAPPING_KEY_STATE:
		return yaml_parser_parse_block_mapping_key(parser, event, false)

	case yaml_PARSE_BLOCK_MAPPING_VALUE_STATE:
		return yaml_parser_parse_block_mapping_value(parser, event)

	case yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE:
		return yaml_parser_parse_flow_sequence_entry(parser, event, true)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_flow_sequence_entry(parser, event, false)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_key(parser, event)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_value(parser, event)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_end(parser, event)

	case yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE:
		return yaml_parser_parse_flow_mapping_key(parser, event, true)

	case yaml_PARSE_FLOW_MAPPING_KEY_STATE:
		return yaml_parser_parse_flow_mapping_key(parser, event, false)

	case yaml_PARSE_FLOW_MAPPING_VALUE_STATE:
		return yaml_parser_parse_flow_mapping_value(parser, event, false)

	case yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE:
		return yaml_parser_parse_flow_mapping_value(parser, event, true)

	default:
		panic("invalid parser state")
	}
	return false
}

// Parse the production:
// stream   ::= STREAM-START implicit_document? explicit_document* STREAM-END
//              ************
func yaml_parser_parse_stream_start(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != yaml_STREAM_START_TOKEN {
		return yaml_parser_set_parser_error(parser, "did not find expected <stream-start>", token.start_mark)
	}
	parser.state = yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE
	*event = yaml_event_t{
		typ:        yaml_STREAM_START_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
		encoding:   token.encoding,
	}
	skip_token(parser)
	return true
}

// Parse the productions:
// implicit_document    ::= block_node DOCUMENT-END*
//                          *
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//                          *************************
func yaml_parser_parse_document_start(parser *yaml_parser_t, event *yaml_event_t, implicit bool) bool {

	token := peek_token(parser)
	if token == nil {
		return false
	}

	// Parse extra document end indicators.
	if !implicit {
		for token.typ == yaml_DOCUMENT_END_TOKEN {
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
		}
	}

	if implicit && token.typ != yaml_VERSION_DIRECTIVE_TOKEN &&
		token.typ != yaml_TAG_DIRECTIVE_TOKEN &&
		token.typ != yaml_DOCUMENT_START_TOKEN &&
		token.typ != yaml_STREAM_END_TOKEN {
		// Parse an implicit document.
		if !yaml_parser_process_directives(parser, nil, nil) {
			return false
		}
		parser.states = append(parser.states, yaml_PARSE_DOCUMENT_END_STATE)
		parser.state = yaml_PARSE_BLOCK_NODE_STATE

		*event = yaml_event_t{
			typ:        yaml_DOCUMENT_START_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}

	} else if token.typ != yaml_STREAM_END_TOKEN {
		// Parse an explicit document.
		var version_directive *yaml_version_directive_t
		var tag_directives []yaml_tag_directive_t
		start_mark := token.start_mark
		if !yaml_parser_process_directives(parser, &version_directive, &tag_directives) {
			return false
		}
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_DOCUMENT_START_TOKEN {
			yaml_parser_set_parser_error(parser,
				"did not find expected <document start>", token.start_mark)
			return false
		}
		parser.states = append(parser.states, yaml_PARSE_DOCUMENT_END_STATE)
		parser.state = yaml_PARSE_DOCUMENT_CONTENT_STATE
		end_mark := token.end_mark

		*event = yaml_event_t{
			typ:               yaml_DOCUMENT_START_EVENT,
			start_mark:        start_mark,
			end_mark:          end_mark,
			version_directive: version_directive,
			tag_directives:    tag_directives,
			implicit:          false,
		}
		skip_token(parser)

	} else {
		// Parse the stream end.
		parser.state = yaml_PARSE_END_STATE
		*event = yaml_event_t{
			typ:        yaml_STREAM_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
	}

	return true
}

// Parse the productions:
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//                                                    ***********
//
func yaml_parser_parse_document_content(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ == yaml_VERSION_DIRECTIVE_TOKEN ||
		token.typ == yaml_TAG_DIRECTIVE_TOKEN ||
		token.typ == yaml_DOCUMENT_START_TOKEN ||
		token.typ == yaml_DOCUMENT_END_TOKEN ||
		token.typ == yaml_STREAM_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		return yaml_parser_process_empty_scalar(parser, event,
			token.start_mark)
	}
	return yaml_parser_parse_node(parser, event, true, false)
}

// Parse the productions:
// implicit_document    ::= block_node DOCUMENT-END*
//                                     *************
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//
func yaml_parser_parse_document_end(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	start_mark := token.start_mark
	end_mark := token.start_mark

	implicit := true
	if token.typ == yaml_DOCUMENT_END_TOKEN {
		end_mark = token.end_mark
		skip_token(parser)
		implicit = false
	}

	parser.tag_directives = parser.tag_directives[:0]

	parser.state = yaml_PARSE_DOCUMENT_START_STATE
	*event = yaml_event_t{
		typ:        yaml_DOCUMENT_END_EVENT,
		start_mark: start_mark,
		end_mark:   end_mark,
		implicit:   implicit,
	}
	return true
}

// Parse the productions:
// block_node_or_indentless_sequence    ::=
//                          ALIAS
//                          *****
//                          | properties (block_content | indentless_block_sequence)?
//                            **********  *
//                          | block_content | indentless_block_sequence
//                            *
// block_node           ::= ALIAS
//                          *****
//                          | properties block_content?
//                            ********** *
//                          | block_content
//                            *
// flow_node            ::= ALIAS
//                          *****
//                          | properties flow_content?
//                            ********** *
//                          | flow_content
//                            *
// properties           ::= TAG ANCHOR? | ANCHOR TAG?
//                          *************************
// block_content        ::= block_collection | flow_collection | SCALAR
//                                                               ******
// flow_content         ::= flow_collection | SCALAR
//                                            ******
func yaml_parser_parse_node(parser *yaml_parser_t, event *yaml_event_t, block, indentless_sequence bool) bool {
	//defer trace("yaml_parser_parse_node", "block:", block, "indentless_sequence:", indentless_sequence)()

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ == yaml_ALIAS_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		*event = yaml_event_t{
			typ:        yaml_ALIAS_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
			anchor:     token.value,
		}
		skip_token(parser)
		return true
	}

	start_mark := token.start_mark
	end_mark := token.start_mark

	var tag_token bool
	var tag_handle, tag_suffix, anchor []byte
	var tag_mark yaml_mark_t
	if token.typ == yaml_ANCHOR_TOKEN {
		anchor = token.value
		start_mark = token.start_mark
		end_mark = token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ == yaml_TAG_TOKEN {
			tag_token = true
			tag_handle = token.value
			tag_suffix = token.suffix
			tag_mark = token.start_mark
			end_mark = token.end_mark
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
		}
	} else if token.typ == yaml_TAG_TOKEN {
		tag_token = true
		tag_handle = token.value
		tag_suffix = token.suffix
		start_mark = token.start_mark
		tag_mark = token.start_mark
		end_mark = token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ == yaml_ANCHOR_TOKEN {
			anchor = token.value
			end_mark = token.end_mark
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
		}
	}

	var tag []byte
	if tag_token {
		if len(tag_handle) == 0 {
			tag = tag_suffix
			tag_suffix = nil
		} else {
			for i := range parser.tag_directives {
				if bytes.Equal(parser.tag_directives[i].handle, tag_handle) {
					tag = append([]byte(nil), parser.tag_directives[i].prefix...)
					tag = append(tag, tag_suffix...)
					break
				}
			}
			if len(tag) == 0 {
				yaml_parser_set_parser_error_context(parser,
					"while parsing a node", start_mark,
					"found undefined tag handle", tag_mark)
				return false
			}
		}
	}

	implicit := len(tag) == 0
	if indentless_sequence && token.typ == yaml_BLOCK_ENTRY_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE
		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_BLOCK_SEQUENCE_STYLE),
		}
		return true
	}
	if token.typ == yaml_SCALAR_TOKEN {
		var plain_implicit, quoted_implicit bool
		end_mark = token.end_mark
		if (len(tag) == 0 && token.style == yaml_PLAIN_SCALAR_STYLE) || (len(tag) == 1 && tag[0] == '!') {
			plain_implicit = true
		} else if len(tag) == 0 {
			quoted_implicit = true
		}
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = yaml_event_t{
			typ:             yaml_SCALAR_EVENT,
			start_mark:      start_mark,
			end_mark:        end_mark,
			anchor:          anchor,
			tag:             tag,
			value:           token.value,
			implicit:        plain_implicit,
			quoted_implicit: quoted_implicit,
			style:           yaml_style_t(token.style),
		}
		skip_token(parser)
		return true
	}
	if token.typ == yaml_FLOW_SEQUENCE_START_TOKEN {
		// [Go] Some of the events below can be merged as they differ only on style.
		end_mark = token.end_mark
		parser.state = yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE
		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_FLOW_SEQUENCE_STYLE),
		}
		return true
	}
	if token.typ == yaml_FLOW_MAPPING_START_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE
		*event = yaml_event_t{
			typ:        yaml_MAPPING_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_FLOW_MAPPING_STYLE),
		}
		return true
	}
	if block && token.typ == yaml_BLOCK_SEQUENCE_START_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE
		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_BLOCK_SEQUENCE_STYLE),
		}
		return true
	}
	if block && token.typ == yaml_BLOCK_MAPPING_START_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE
		*event = yaml_event_t{
			typ:        yaml_MAPPING_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_BLOCK_MAPPING_STYLE),
		}
		return true
	}
	if len(anchor) > 0 || len(tag) > 0 {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = yaml_event_t{
			typ:             yaml_SCALAR_EVENT,
			start_mark:      start_mark,
			end_mark:        end_mark,
			anchor:          anchor,
			tag:             tag,
			implicit:        implicit,
			quoted_implicit: false,
			style:           yaml_style_t(yaml_PLAIN_SCALAR_STYLE),
		}
		return true
	}

	context := "while parsing a flow node"
	if block {
		context = "while parsing a block node"
	}
	yaml_parser_set_parser_error_context(parser, context, start_mark,
		"did not find expected node content", token.start_mark)
	return false
}

// Parse the productions:
// block_sequence ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
//                    ********************  *********** *             *********
//
func yaml_parser_parse_block_sequence_entry(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ == yaml_BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_BLOCK_ENTRY_TOKEN && token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, true, false)
		} else {
			parser.state = yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	}
	if token.typ == yaml_BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]

		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}

		skip_token(parser)
		return true
	}

	context_mark := parser.marks[len(parser.marks)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	return yaml_parser_set_parser_error_context(parser,
		"while parsing a block collection", context_mark,
		"did not find expected '-' indicator", token.start_mark)
}

// Parse the productions:
// indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
//                           *********** *
func yaml_parser_parse_indentless_sequence_entry(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ == yaml_BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_BLOCK_ENTRY_TOKEN &&
			token.typ != yaml_KEY_TOKEN &&
			token.typ != yaml_VALUE_TOKEN &&
			token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, true, false)
		}
		parser.state = yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE
		return yaml_parser_process_empty_scalar(parser, event, mark)
	}
	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]

	*event = yaml_event_t{
		typ:        yaml_SEQUENCE_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.start_mark, // [Go] Shouldn't this be token.end_mark?
	}
	return true
}

// Parse the productions:
// block_mapping        ::= BLOCK-MAPPING_START
//                          *******************
//                          ((KEY block_node_or_indentless_sequence?)?
//                            *** *
//                          (VALUE block_node_or_indentless_sequence?)?)*
//
//                          BLOCK-END
//                          *********
//
func yaml_parser_parse_block_mapping_key(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ == yaml_KEY_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_KEY_TOKEN &&
			token.typ != yaml_VALUE_TOKEN &&
			token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_BLOCK_MAPPING_VALUE_STATE)
			return yaml_parser_parse_node(parser, event, true, true)
		} else {
			parser.state = yaml_PARSE_BLOCK_MAPPING_VALUE_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	} else if token.typ == yaml_BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]
		*event = yaml_event_t{
			typ:        yaml_MAPPING_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
		return true
	}

	context_mark := parser.marks[len(parser.marks)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	return yaml_parser_set_parser_error_context(parser,
		"while parsing a block mapping", context_mark,
		"did not find expected key", token.start_mark)
}

// Parse the productions:
// block_mapping        ::= BLOCK-MAPPING_START
//
//                          ((KEY block_node_or_indentless_sequence?)?
//
//                          (VALUE block_node_or_indentless_sequence?)?)*
//                           ***** *
//                          BLOCK-END
//
//
func yaml_parser_parse_block_mapping_value(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ == yaml_VALUE_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_KEY_TOKEN &&
			token.typ != yaml_VALUE_TOKEN &&
			token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_BLOCK_MAPPING_KEY_STATE)
			return yaml_parser_parse_node(parser, event, true, true)
		}
		parser.state = yaml_PARSE_BLOCK_MAPPING_KEY_STATE
		return yaml_parser_process_empty_scalar(parser, event, mark)
	}
	parser.state = yaml_PARSE_BLOCK_MAPPING_KEY_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

// Parse the productions:
// flow_sequence        ::= FLOW-SEQUENCE-START
//                          *******************
//                          (flow_sequence_entry FLOW-ENTRY)*
//                           *                   **********
//                          flow_sequence_entry?
//                          *
//                          FLOW-SEQUENCE-END
//                          *****************
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                          *
//
func yaml_parser_parse_flow_sequence_entry(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
		if !first {
			if token.typ == yaml_FLOW_ENTRY_TOKEN {
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}
			} else {
				context_mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]
				return yaml_parser_set_parser_error_context(parser,
					"while parsing a flow sequence", context_mark,
					"did not find expected ',' or ']'", token.start_mark)
			}
		}

		if token.typ == yaml_KEY_TOKEN {
			parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE
			*event = yaml_event_t{
				typ:        yaml_MAPPING_START_EVENT,
				start_mark: token.start_mark,
				end_mark:   token.end_mark,
				implicit:   true,
				style:      yaml_style_t(yaml_FLOW_MAPPING_STYLE),
			}
			skip_token(parser)
			return true
		} else if token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]

	*event = yaml_event_t{
		typ:        yaml_SEQUENCE_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}

	skip_token(parser)
	return true
}

//
// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                      *** *
//
func yaml_parser_parse_flow_sequence_entry_mapping_key(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != yaml_VALUE_TOKEN &&
		token.typ != yaml_FLOW_ENTRY_TOKEN &&
		token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
		parser.states = append(parser.states, yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE)
		return yaml_parser_parse_node(parser, event, false, false)
	}
	mark := token.end_mark
	skip_token(parser)
	parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE
	return yaml_parser_process_empty_scalar(parser, event, mark)
}

// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                                      ***** *
//
func yaml_parser_parse_flow_sequence_entry_mapping_value(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ == yaml_VALUE_TOKEN {
		skip_token(parser)
		token := peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_FLOW_ENTRY_TOKEN && token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}
	parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                                                      *
//
func yaml_parser_parse_flow_sequence_entry_mapping_end(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE
	*event = yaml_event_t{
		typ:        yaml_MAPPING_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.start_mark, // [Go] Shouldn't this be end_mark?
	}
	return true
}

// Parse the productions:
// flow_mapping         ::= FLOW-MAPPING-START
//                          ******************
//                          (flow_mapping_entry FLOW-ENTRY)*
//                           *                  **********
//                          flow_mapping_entry?
//                          ******************
//                          FLOW-MAPPING-END
//                          ****************
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                          *           *** *
//
func yaml_parser_parse_flow_mapping_key(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ != yaml_FLOW_MAPPING_END_TOKEN {
		if !first {
			if token.typ == yaml_FLOW_ENTRY_TOKEN {
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}
			} else {
				context_mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]
				return yaml_parser_set_parser_error_context(parser,
					"while parsing a flow mapping", context_mark,
					"did not find expected ',' or '}'", token.start_mark)
			}
		}

		if token.typ == yaml_KEY_TOKEN {
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
			if token.typ != yaml_VALUE_TOKEN &&
				token.typ != yaml_FLOW_ENTRY_TOKEN &&
				token.typ != yaml_FLOW_MAPPING_END_TOKEN {
				parser.states = append(parser.states, yaml_PARSE_FLOW_MAPPING_VALUE_STATE)
				return yaml_parser_parse_node(parser, event, false, false)
			} else {
				parser.state = yaml_PARSE_FLOW_MAPPING_VALUE_STATE
				return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
			}
		} else if token.typ != yaml_FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	*event = yaml_event_t{
		typ:        yaml_MAPPING_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}
	skip_token(parser)
	return true
}

// Parse the productions:
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                   *                  ***** *
//
func yaml_parser_parse_flow_mapping_value(parser *yaml_parser_t, event *yaml_event_t, empty bool) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if empty {
		parser.state = yaml_PARSE_FLOW_MAPPING_KEY_STATE
		return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
	}
	if token.typ == yaml_VALUE_TOKEN {
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_FLOW_ENTRY_TOKEN && token.typ != yaml_FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_MAPPING_KEY_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}
	parser.state = yaml_PARSE_FLOW_MAPPING_KEY_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

// Generate an empty scalar event.
func yaml_parser_process_empty_scalar(parser *yaml_parser_t, event *yaml_event_t, mark yaml_mark_t) bool {
	*event = yaml_event_t{
		typ:        yaml_SCALAR_EVENT,
		start_mark: mark,
		end_mark:   mark,
		value:      nil, // Empty
		implicit:   true,
		style:      yaml_style_t(yaml_PLAIN_SCALAR_STYLE),
	}
	return true
}

var default_tag_directives = []yaml_tag_directive_t{
	{[]byte("!"), []byte("!")},
	{[]byte("!!"), []byte("tag:yaml.org,2002:")},
}

// Parse directives.
func yaml_parser_process_directives(parser *yaml_parser_t,
	version_directive_ref **yaml_version_directive_t,
	tag_directives_ref *[]yaml_tag_directive_t) bool {

	var version_directive *yaml_version_directive_t
	var tag_directives []yaml_tag_directive_t

	token := peek_token(parser)
	if token == nil {
		return false
	}

	for token.typ == yaml_VERSION_DIRECTIVE_TOKEN || token.typ == yaml_TAG_DIRECTIVE_TOKEN {
		if token.typ == yaml_VERSION_DIRECTIVE_TOKEN {
			if version_directive != nil {
				yaml_parser_set_parser_error(parser,
					"found duplicate %YAML directive", token.start_mark)
				return false
			}
			if token.major != 1 || token.minor != 1 {
				yaml_parser_set_parser_error(parser,
					"found incompatible YAML document", token.start_mark)
				return false
			}
			version_directive = &yaml_version_directive_t{
				major: token.major,
				minor: token.minor,
			}
		} else if token.typ == yaml_TAG_DIRECTIVE_TOKEN {
			value := yaml_tag_directive_t{
				handle: token.value,
				prefix: token.prefix,
			}
			if !yaml_parser_append_tag_directive(parser, value, false, token.start_mark) {
				return false
			}
			tag_directives = append(tag_directives, value)
		}

		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
	}

	for i := range default_tag_directives {
		if !yaml_parser_append_tag_directive(parser, default_tag_directives[i], true, token.start_mark) {
			return false
		}
	}

	if version_directive_ref != nil {
		*version_directive_ref = version_directive
	}
	if tag_directives_ref != nil {
		*tag_directives_ref = tag_directives
	}
	return true
}

// Append a tag directive to the directives stack.
func yaml_parser_append_tag_directive(parser *yaml_parser_t, value yaml_tag_directive_t, allow_duplicates bool, mark yaml_mark_t) bool {
	for i := range parser.tag_directives {
		if bytes.Equal(value.handle, parser.tag_directives[i].handle) {
			if allow_duplicates {
				return true
			}
			return yaml_parser_set_parser_error(parser, "found duplicate %TAG directive", mark)
		}
	}

	// [Go] I suspect the copy is unnecessary. This was likely done
	// because there was no way to track ownership of the data.
	value_copy := yaml_tag_directive_t{
		handle: make([]byte, len(value.handle)),
		prefix: make([]byte, len(value.prefix)),
	}
	copy(value_copy.handle, value.handle)
	copy(value_copy.prefix, value.prefix)
	parser.tag_directives = append(parser.tag_directives, value_copy)
	return true
}
