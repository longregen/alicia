package encoding

import (
	"encoding/binary"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	// Register decoder for msgpack extension type 0 (timestamp)
	// JavaScript's msgpackr library encodes Date objects as extension type 0
	// Format: 8 bytes - first 4 bytes are seconds (big-endian), last 4 are nanoseconds
	// Or: 12 bytes - first 4 bytes are nanoseconds, last 8 are seconds (for dates after 2106)
	msgpack.RegisterExtDecoder(0, time.Time{}, func(dec *msgpack.Decoder, v reflect.Value, extLen int) error {
		data := make([]byte, extLen)
		if _, err := dec.Buffered().Read(data); err != nil {
			return err
		}

		var t time.Time
		switch extLen {
		case 4:
			// 32-bit format: seconds only
			secs := int64(binary.BigEndian.Uint32(data))
			t = time.Unix(secs, 0)
		case 8:
			// 64-bit format: nanoseconds in upper 30 bits, seconds in lower 34 bits
			val := binary.BigEndian.Uint64(data)
			nsec := int64(val >> 34)
			sec := int64(val & 0x3ffffffff)
			t = time.Unix(sec, nsec)
		case 12:
			// 96-bit format: 4 bytes nanoseconds + 8 bytes seconds
			nsec := int64(binary.BigEndian.Uint32(data[:4]))
			sec := int64(binary.BigEndian.Uint64(data[4:]))
			t = time.Unix(sec, nsec)
		default:
			// Fallback: treat as milliseconds (common in JS)
			if extLen == 8 {
				ms := int64(binary.BigEndian.Uint64(data))
				t = time.UnixMilli(ms)
			} else {
				t = time.Time{}
			}
		}
		v.Set(reflect.ValueOf(t))
		return nil
	})

	// Also register for int64 (timestamps as milliseconds)
	msgpack.RegisterExtDecoder(0, int64(0), func(dec *msgpack.Decoder, v reflect.Value, extLen int) error {
		data := make([]byte, extLen)
		if _, err := dec.Buffered().Read(data); err != nil {
			return err
		}

		var ms int64
		switch extLen {
		case 4:
			ms = int64(binary.BigEndian.Uint32(data)) * 1000
		case 8:
			val := binary.BigEndian.Uint64(data)
			sec := int64(val & 0x3ffffffff)
			ms = sec * 1000
		case 12:
			sec := int64(binary.BigEndian.Uint64(data[4:]))
			ms = sec * 1000
		default:
			ms = 0
		}
		v.SetInt(ms)
		return nil
	})
}

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
