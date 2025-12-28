package encoding

import (
	"net/http"
	"strings"

	"github.com/vmihailenco/msgpack/v5"
)

const ContentTypeMsgpack = "application/msgpack"
const ContentTypeJSON = "application/json"

// NegotiateContentType checks the Accept header and returns the preferred content type
func NegotiateContentType(r *http.Request) string {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return ContentTypeJSON
	}

	// Check if MessagePack is explicitly requested
	if strings.Contains(accept, ContentTypeMsgpack) {
		return ContentTypeMsgpack
	}

	// Check for */* wildcard
	if strings.Contains(accept, "*/*") {
		return ContentTypeJSON
	}

	// Default to JSON
	return ContentTypeJSON
}

// WriteMsgpack writes a MessagePack response with the given status code
func WriteMsgpack(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", ContentTypeMsgpack)
	w.WriteHeader(status)

	encoder := msgpack.NewEncoder(w)
	return encoder.Encode(data)
}

// ReadMsgpack reads MessagePack data from the request body
func ReadMsgpack(r *http.Request, target interface{}) error {
	decoder := msgpack.NewDecoder(r.Body)
	return decoder.Decode(target)
}
