package xmldom

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// Unmarshal parses XML-encoded data and stores the result in the value pointed to by v.
// This function delegates to Go's standard xml.Unmarshal for struct unmarshaling.
func Unmarshal(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}

// UnmarshalDOM parses XML-encoded data and returns a DOM Document.
// This creates a DOM tree that can be manipulated using the xmldom API.
func UnmarshalDOM(data []byte) (Document, error) {
	decoder := NewDecoder(strings.NewReader(string(data)))
	return decoder.Decode()
}

// Marshal returns the XML encoding of v.
// This function handles both DOM Documents and regular structs.
func Marshal(v interface{}) ([]byte, error) {
	// Check if v is a DOM Document
	if doc, ok := v.(Document); ok {
		return marshalDOM(doc, "", "", true)
	}
	// Check if v is a DOM Element
	if elem, ok := v.(Element); ok {
		return marshalElement(elem, "", "", true)
	}
	// Check if v is any DOM Node
	if node, ok := v.(Node); ok {
		return marshalNode(node, "", "", true)
	}
	// For non-DOM objects, delegate to Go's standard xml.Marshal
	return xml.Marshal(v)
}

// MarshalIndent returns the XML encoding of v with optional prefix and indent.
// If prefix or indent is provided, the output will include newlines and indentation.
func MarshalIndent(v interface{}, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	return MarshalIndentWithOptions(v, prefix, indent, preserveWhitespace)
}

// MarshalIndentWithOptions returns the XML encoding of v with optional prefix, indent, and whitespace preservation.
// If preserveWhitespace is true, newlines and tabs in text content will not be escaped as HTML entities.
func MarshalIndentWithOptions(v interface{}, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	// Check if v is a DOM Document
	if doc, ok := v.(Document); ok {
		return marshalDOMWithOptions(doc, prefix, indent, preserveWhitespace)
	}
	// Check if v is a DOM Element
	if elem, ok := v.(Element); ok {
		return marshalElementWithOptions(elem, prefix, indent, preserveWhitespace)
	}
	// Check if v is any DOM Node
	if node, ok := v.(Node); ok {
		return marshalNodeWithOptions(node, prefix, indent, preserveWhitespace)
	}
	// For non-DOM objects, delegate to Go's standard xml.MarshalIndent
	return xml.MarshalIndent(v, prefix, indent)
}

// marshalDOM serializes a DOM Document to XML
func marshalDOM(doc Document, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	return marshalDOMWithOptions(doc, prefix, indent, preserveWhitespace)
}

// marshalDOMWithOptions serializes a DOM Document to XML with whitespace preservation option
func marshalDOMWithOptions(doc Document, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	var buf bytes.Buffer

	// Write XML declaration
	buf.WriteString(`<?xml version="1.0"?>`)

	// If indent is provided, add a newline after declaration
	if indent != "" {
		buf.WriteString("\n")
	}

	// Serialize the document element
	root := doc.DocumentElement()
	if root == nil {
		return buf.Bytes(), nil // Empty document
	}

	if err := serializeElementWithOptions(&buf, root, false, prefix, indent, 0, preserveWhitespace); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// marshalElement serializes a DOM Element to XML (without XML declaration)
func marshalElement(elem Element, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	return marshalElementWithOptions(elem, prefix, indent, preserveWhitespace)
}

// marshalElementWithOptions serializes a DOM Element to XML (without XML declaration) with whitespace preservation option
func marshalElementWithOptions(elem Element, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	var buf bytes.Buffer
	if err := serializeElementWithOptions(&buf, elem, false, prefix, indent, 0, preserveWhitespace); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// marshalNode serializes any DOM Node to XML (without XML declaration)
func marshalNode(node Node, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	return marshalNodeWithOptions(node, prefix, indent, false)
}

// marshalNodeWithOptions serializes any DOM Node to XML (without XML declaration) with whitespace preservation option
func marshalNodeWithOptions(node Node, prefix, indent string, preserveWhitespace bool) ([]byte, error) {
	var buf bytes.Buffer
	if err := serializeNodeWithOptions(&buf, node, prefix, indent, 0, preserveWhitespace); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// serializeElement serializes an Element and its children to XML
func serializeElement(buf *bytes.Buffer, elem Element, skipRoot bool, prefix, indent string, depth int) error {
	// Write indentation if indent is provided
	if indent != "" && !skipRoot {
		buf.WriteString(strings.Repeat(indent, depth))
	}

	if !skipRoot {
		// Write opening tag
		buf.WriteString("<")
		buf.WriteString(string(elem.TagName()))

		// Write attributes
		attrs := elem.Attributes()
		if attrs != nil {
			for i := uint(0); i < attrs.Length(); i++ {
				attr := attrs.Item(i)
				if attr != nil && attr.NodeType() == ATTRIBUTE_NODE {
					if attrNode, ok := attr.(Attr); ok {
						buf.WriteString(" ")
						buf.WriteString(string(attrNode.Name()))

						// Use single quotes if the value contains double quotes (e.g., JSON)
						// This makes JSON attributes much more readable
						attrValue := string(attrNode.Value())
						useSingleQuote := strings.Contains(attrValue, `"`)

						if useSingleQuote {
							buf.WriteString(`='`)
							buf.WriteString(attrValue)
							buf.WriteString(`'`)
						} else {
							buf.WriteString(`="`)
							buf.WriteString(EscapeString(attrValue))
							buf.WriteString(`"`)
						}
					}
				}
			}
		}

		// Check if element has children
		hasChildren := elem.HasChildNodes()
		if !hasChildren {
			// For SCXML conformance, always use explicit opening/closing tags
			// instead of self-closing tags for empty elements
			buf.WriteString("></")
			buf.WriteString(string(elem.TagName()))
			buf.WriteString(">")
			if indent != "" {
				buf.WriteString("\n")
			}
			return nil
		}

		buf.WriteString(">")
		if indent != "" && hasChildren {
			buf.WriteString("\n")
		}
	}

	// Serialize children
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if err := serializeNode(buf, child, prefix, indent, depth+1); err != nil {
			return err
		}
	}

	if !skipRoot {
		// Write indentation for closing tag if indent is provided
		if indent != "" {
			buf.WriteString(strings.Repeat(indent, depth))
		}
		// Write closing tag
		buf.WriteString("</")
		buf.WriteString(string(elem.TagName()))
		buf.WriteString(">")
		if indent != "" {
			buf.WriteString("\n")
		}
	}

	return nil
}

// serializeElementWithOptions serializes an Element and its children to XML with whitespace preservation option
func serializeElementWithOptions(buf *bytes.Buffer, elem Element, skipRoot bool, prefix, indent string, depth int, preserveWhitespace bool) error {
	// Write indentation if indent is provided
	if indent != "" && !skipRoot {
		buf.WriteString(strings.Repeat(indent, depth))
	}

	if !skipRoot {
		// Write opening tag
		buf.WriteString("<")
		buf.WriteString(string(elem.TagName()))

		// Write attributes
		attrs := elem.Attributes()
		if attrs != nil {
			for i := uint(0); i < attrs.Length(); i++ {
				attr := attrs.Item(i)
				if attr != nil && attr.NodeType() == ATTRIBUTE_NODE {
					if attrNode, ok := attr.(Attr); ok {
						buf.WriteString(" ")
						buf.WriteString(string(attrNode.Name()))

						// Use single quotes if the value contains double quotes (e.g., JSON)
						// This makes JSON attributes much more readable
						attrValue := string(attrNode.Value())
						useSingleQuote := strings.Contains(attrValue, `"`)

						if useSingleQuote {
							buf.WriteString(`='`)
							// When using single quotes, escape any single quotes in the value
							if !preserveWhitespace {
								buf.WriteString(strings.ReplaceAll(EscapeString(attrValue), `"`, `"`))
							} else {
								buf.WriteString(attrValue)
							}
							buf.WriteString(`'`)
						} else {
							buf.WriteString(`="`)
							if !preserveWhitespace {
								buf.WriteString(EscapeString(attrValue))
							} else {
								buf.WriteString(attrValue)
							}
							buf.WriteString(`"`)
						}
					}
				}
			}
		}

		// Check if element has children
		hasChildren := elem.HasChildNodes()
		if !hasChildren {
			// For SCXML conformance, always use explicit opening/closing tags
			// instead of self-closing tags for empty elements
			buf.WriteString("></")
			buf.WriteString(string(elem.TagName()))
			buf.WriteString(">")
			if indent != "" {
				buf.WriteString("\n")
			}
			return nil
		}

		buf.WriteString(">")
		if indent != "" && hasChildren {
			buf.WriteString("\n")
		}
	}

	// Serialize children
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if err := serializeNodeWithOptions(buf, child, prefix, indent, depth+1, preserveWhitespace); err != nil {
			return err
		}
	}

	if !skipRoot {
		// Write indentation for closing tag if indent is provided
		if indent != "" {
			buf.WriteString(strings.Repeat(indent, depth))
		}
		// Write closing tag
		buf.WriteString("</")
		buf.WriteString(string(elem.TagName()))
		buf.WriteString(">")
		if indent != "" {
			buf.WriteString("\n")
		}
	}

	return nil
}

// serializeNode serializes any DOM node to XML
func serializeNode(buf *bytes.Buffer, node Node, prefix, indent string, depth int) error {
	switch node.NodeType() {
	case ELEMENT_NODE:
		if elem, ok := node.(Element); ok {
			return serializeElement(buf, elem, false, prefix, indent, depth)
		}
	case TEXT_NODE:
		if text, ok := node.(Text); ok {
			// Skip whitespace-only text nodes when indenting
			textData := string(text.Data())
			if indent != "" && strings.TrimSpace(textData) == "" {
				return nil
			}
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			buf.WriteString(EscapeString(textData))
			if indent != "" {
				buf.WriteString("\n")
			}
		}
	case COMMENT_NODE:
		if comment, ok := node.(Comment); ok {
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			buf.WriteString("<!--")
			buf.WriteString(string(comment.Data()))
			buf.WriteString("-->")
			if indent != "" {
				buf.WriteString("\n")
			}
		}
	case CDATA_SECTION_NODE:
		if cdata, ok := node.(CDATASection); ok {
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			buf.WriteString("<![CDATA[")
			buf.WriteString(string(cdata.Data()))
			buf.WriteString("]]>")
			if indent != "" {
				buf.WriteString("\n")
			}
		}
	case PROCESSING_INSTRUCTION_NODE:
		if pi, ok := node.(ProcessingInstruction); ok {
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			buf.WriteString("<?")
			buf.WriteString(string(pi.Target()))
			if data := string(pi.Data()); data != "" {
				buf.WriteString(" ")
				buf.WriteString(data)
			}
			buf.WriteString("?>")
			if indent != "" {
				buf.WriteString("\n")
			}
		}
		// Skip other node types for now
	}
	return nil
}

// serializeNodeWithOptions serializes any DOM node to XML with whitespace preservation option
func serializeNodeWithOptions(buf *bytes.Buffer, node Node, prefix, indent string, depth int, preserveWhitespace bool) error {
	switch node.NodeType() {
	case ELEMENT_NODE:
		if elem, ok := node.(Element); ok {
			return serializeElementWithOptions(buf, elem, false, prefix, indent, depth, preserveWhitespace)
		}
	case TEXT_NODE:
		if text, ok := node.(Text); ok {
			textData := string(text.Data())
			// Skip whitespace-only text nodes when indenting
			if indent != "" && strings.TrimSpace(textData) == "" {
				return nil
			}
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			if preserveWhitespace {
				// Write text content without escaping whitespace characters
				buf.WriteString(textData)
			} else {
				// Use standard escaping for XML compliance
				buf.WriteString(EscapeString(textData))
			}
			if indent != "" {
				buf.WriteString("\n")
			}
		}
	case COMMENT_NODE:
		if comment, ok := node.(Comment); ok {
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			buf.WriteString("<!--")
			if preserveWhitespace {
				buf.WriteString(string(comment.Data()))
			} else {
				buf.WriteString(EscapeString(string(comment.Data())))
			}
			buf.WriteString("-->")
			if indent != "" {
				buf.WriteString("\n")
			}
		}
	case CDATA_SECTION_NODE:
		if cdata, ok := node.(CDATASection); ok {
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			buf.WriteString("<![CDATA[")
			buf.WriteString(string(cdata.Data()))
			buf.WriteString("]]>")
			if indent != "" {
				buf.WriteString("\n")
			}
		}
	case PROCESSING_INSTRUCTION_NODE:
		if pi, ok := node.(ProcessingInstruction); ok {
			if indent != "" {
				buf.WriteString(strings.Repeat(indent, depth))
			}
			buf.WriteString("<?")
			buf.WriteString(string(pi.Target()))
			if data := string(pi.Data()); data != "" {
				buf.WriteString(" ")
				buf.WriteString(data)
			}
			buf.WriteString("?>")
			if indent != "" {
				buf.WriteString("\n")
			}
		}
		// Skip other node types for now
	}
	return nil
}
