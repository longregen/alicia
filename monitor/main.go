package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

// ANSI colors
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
	bgGray  = "\033[48;5;236m"
)

// monitorFrame wraps each message with routing metadata
type monitorFrame struct {
	Src  string `msgpack:"src"`
	Dst  string `msgpack:"dst"`
	Data []byte `msgpack:"data"`
}

// Envelope mirrors the protocol envelope for decoding
type Envelope struct {
	ConversationID string      `msgpack:"conversationId,omitempty"`
	Type           uint16      `msgpack:"type"`
	Body           interface{} `msgpack:"body"`
	TraceID        string      `msgpack:"trace_id,omitempty"`
	SpanID         string      `msgpack:"span_id,omitempty"`
	TraceFlags     byte        `msgpack:"trace_flags,omitempty"`
	SessionID      string      `msgpack:"session_id,omitempty"`
	UserID         string      `msgpack:"user_id,omitempty"`
}

var typeNames = map[uint16]string{
	1:  "Error",
	2:  "UserMessage",
	3:  "AssistantMsg",
	5:  "ReasoningStep",
	6:  "ToolUseRequest",
	7:  "ToolUseResult",
	8:  "Ack",
	13: "StartAnswer",
	14: "MemoryTrace",
	16: "AssistantSentence",
	33: "GenRequest",
	34: "ThinkingSummary",
	35: "TitleUpdate",
	40: "Subscribe",
	41: "Unsubscribe",
	42: "SubscribeAck",
	43: "UnsubscribeAck",
	50: "BranchUpdate",
	51: "VoiceJoinRequest",
	52: "VoiceJoinAck",
	53: "VoiceLeaveRequest",
	54: "VoiceLeaveAck",
	55: "VoiceStatus",
	56: "VoiceSpeaking",
	60: "PreferencesUpdate",
	70: "AssistantToolsRegister",
	71: "AssistantToolsAck",
	72: "AssistantHeartbeat",
	80: "GenerationComplete",
}

var typeColors = map[uint16]string{
	1:  red,
	2:  green,
	3:  cyan,
	5:  magenta,
	6:  yellow,
	7:  yellow,
	8:  dim,
	13: blue,
	14: magenta,
	16: cyan,
	33: green,
	34: magenta,
	35: blue,
	40: blue,
	41: blue,
	42: blue,
	43: blue,
	50: yellow,
	51: green,
	52: green,
	53: red,
	54: red,
	55: cyan,
	56: cyan,
	60: magenta,
	70: yellow,
	71: yellow,
	72: dim,
	80: green,
}

type rawMessage struct {
	data []byte
	src  string
	dst  string
	ts   time.Time
}

func main() {
	url := flag.String("url", "ws://localhost:8090/api/v1/ws", "WebSocket URL")
	secret := flag.String("secret", "", "Agent secret for auth (optional)")
	flag.Parse()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Printf("%s%s╔══════════════════════════════════════╗%s\n", bold, blue, reset)
	fmt.Printf("%s%s║     Alicia WebSocket Monitor         ║%s\n", bold, blue, reset)
	fmt.Printf("%s%s╚══════════════════════════════════════╝%s\n", bold, blue, reset)
	fmt.Printf("%sConnecting to: %s%s%s\n", dim, reset, *url, reset)

	headers := http.Header{}
	if *secret != "" {
		headers.Set("Authorization", "Bearer "+*secret)
	}

	delays := []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second,
	}

	msgNum := 0
	for {
		conn, err := dialWithRetry(*url, headers, delays, interrupt)
		if err != nil {
			fmt.Printf("\n%s%s─── interrupted ───%s\n", dim, yellow, reset)
			return
		}

		fmt.Printf("%s%s✓ Connected%s\n", bold, green, reset)

		// Subscribe in monitor mode
		subEnv := Envelope{
			Type: 40, // TypeSubscribe
			Body: map[string]interface{}{"monitorMode": true},
		}
		subData, err := msgpack.Marshal(&subEnv)
		if err != nil {
			log.Fatalf("%s✗ Failed to encode subscribe: %v%s\n", red, err, reset)
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, subData); err != nil {
			conn.Close()
			fmt.Printf("%s✗ Failed to send subscribe: %v%s\n", red, err, reset)
			fmt.Printf("%s%s─── reconnecting... ───%s\n", dim, yellow, reset)
			continue
		}
		fmt.Printf("%s%s✓ Subscribed (monitor mode)%s\n\n", bold, green, reset)

		// Receiver goroutine
		msgCh := make(chan rawMessage, 256)
		go func() {
			defer close(msgCh)
			for {
				_, raw, err := conn.ReadMessage()
				if err != nil {
					fmt.Printf("\n%s✗ Read error: %v%s\n", red, err, reset)
					return
				}
				var frame monitorFrame
				if err := msgpack.Unmarshal(raw, &frame); err != nil || len(frame.Data) == 0 {
					msgCh <- rawMessage{data: raw, ts: time.Now()}
					continue
				}
				msgCh <- rawMessage{data: frame.Data, src: frame.Src, dst: frame.Dst, ts: time.Now()}
			}
		}()

		// Printer loop — breaks on disconnect or interrupt
		disconnected := false
		for !disconnected {
			select {
			case msg, ok := <-msgCh:
				if !ok {
					disconnected = true
				} else {
					msgNum++
					printMessage(msgNum, msg)
				}
			case <-interrupt:
				fmt.Printf("\n%s%s─── interrupted ───%s\n", dim, yellow, reset)
				conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				conn.Close()
				return
			}
		}

		conn.Close()
		fmt.Printf("%s%s─── connection lost, reconnecting... ───%s\n\n", dim, yellow, reset)
	}
}

func dialWithRetry(url string, headers http.Header, delays []time.Duration, interrupt <-chan os.Signal) (*websocket.Conn, error) {
	for attempt := 0; ; attempt++ {
		conn, _, err := websocket.DefaultDialer.Dial(url, headers)
		if err == nil {
			return conn, nil
		}
		if attempt >= len(delays) {
			return nil, fmt.Errorf("failed after %d attempts: %w", attempt+1, err)
		}
		fmt.Printf("%s  retrying in %v...%s\n", dim, delays[attempt], reset)
		select {
		case <-time.After(delays[attempt]):
		case <-interrupt:
			return nil, fmt.Errorf("interrupted")
		}
	}
}

func printMessage(num int, msg rawMessage) {
	timestamp := msg.ts.Format("15:04:05.000")

	// Try to decode as protocol envelope
	var env Envelope
	err := msgpack.Unmarshal(msg.data, &env)

	if err == nil && env.Type > 0 {
		printParsedMessage(num, timestamp, &env, msg.data, msg.src, msg.dst)
	} else {
		printRawMessage(num, timestamp, msg.data, err)
	}

	fmt.Println()
}

var routeColors = map[string]string{
	"client": green,
	"agent":  cyan,
	"voice":  magenta,
	"server": yellow,
}

func formatRoute(src, dst string) string {
	if src == "" && dst == "" {
		return ""
	}
	srcColor := routeColors[src]
	dstColor := routeColors[dst]
	if srcColor == "" {
		srcColor = white
	}
	if dstColor == "" {
		dstColor = white
	}
	return fmt.Sprintf("%s%s%s %s→%s %s%s%s", srcColor, src, reset, dim, reset, dstColor, dst, reset)
}

func printParsedMessage(num int, timestamp string, env *Envelope, raw []byte, src, dst string) {
	typeName := typeNames[env.Type]
	if typeName == "" {
		typeName = fmt.Sprintf("Unknown(%d)", env.Type)
	}
	color := typeColors[env.Type]
	if color == "" {
		color = white
	}

	// Header line
	route := formatRoute(src, dst)
	fmt.Printf("%s%s#%d%s %s%s%s %s%s%s",
		dim, bgGray, num, reset,
		dim, timestamp, reset,
		bold, color, typeName)

	if route != "" {
		fmt.Printf(" %s", route)
	}

	if env.ConversationID != "" {
		short := env.ConversationID
		if len(short) > 8 {
			short = short[:8]
		}
		fmt.Printf(" %s[%s]%s", dim, short, reset)
	}
	fmt.Printf("%s\n", reset)

	// Trace context
	if env.TraceID != "" {
		fmt.Printf("  %strace:%s %s", dim, reset, env.TraceID[:16])
		if env.SpanID != "" {
			fmt.Printf(" %sspan:%s %s", dim, reset, env.SpanID)
		}
		fmt.Println()
	}

	// Body
	if env.Body != nil {
		printBody(env.Type, env.Body)
	}

	// Size
	fmt.Printf("  %s(%d bytes)%s\n", dim, len(raw), reset)
}

func printBody(msgType uint16, body interface{}) {
	bodyMap, ok := body.(map[string]interface{})
	if !ok {
		// Fallback: JSON marshal the body
		data, err := json.MarshalIndent(body, "  ", "  ")
		if err == nil && string(data) != "null" {
			fmt.Printf("  %s\n", string(data))
		}
		return
	}

	// Type-specific rendering
	switch msgType {
	case 2: // UserMessage
		if content, ok := bodyMap["content"].(string); ok {
			fmt.Printf("  %s▶%s %s\n", green, reset, truncate(content, 120))
		}
	case 3: // AssistantMsg
		if content, ok := bodyMap["content"].(string); ok {
			fmt.Printf("  %s◀%s %s\n", cyan, reset, truncate(content, 120))
		}
	case 16: // AssistantSentence
		seq := bodyMap["sequence"]
		text, _ := bodyMap["text"].(string)
		isFinal, _ := bodyMap["isFinal"].(bool)
		marker := "…"
		if isFinal {
			marker = "■"
		}
		fmt.Printf("  %s%s%s [%v] %s\n", cyan, marker, reset, seq, truncate(text, 100))
	case 6: // ToolUseRequest
		tool, _ := bodyMap["toolName"].(string)
		fmt.Printf("  %s⚙%s  %s%s%s", yellow, reset, bold, tool, reset)
		if args, ok := bodyMap["arguments"].(map[string]interface{}); ok {
			data, _ := json.Marshal(args)
			fmt.Printf(" %s%s%s", dim, truncate(string(data), 80), reset)
		}
		fmt.Println()
	case 7: // ToolUseResult
		success, _ := bodyMap["success"].(bool)
		if success {
			fmt.Printf("  %s✓%s ", green, reset)
		} else {
			fmt.Printf("  %s✗%s ", red, reset)
			if errMsg, ok := bodyMap["error"].(string); ok {
				fmt.Printf("%s%s%s ", red, truncate(errMsg, 80), reset)
			}
		}
		if result := bodyMap["result"]; result != nil {
			data, _ := json.Marshal(result)
			fmt.Printf("%s%s%s", dim, truncate(string(data), 80), reset)
		}
		fmt.Println()
	case 14: // MemoryTrace
		content, _ := bodyMap["content"].(string)
		relevance := bodyMap["relevance"]
		fmt.Printf("  %s🧠%s %s %s(rel: %v)%s\n", magenta, reset, truncate(content, 80), dim, relevance, reset)
	case 34: // ThinkingSummary
		content, _ := bodyMap["content"].(string)
		progress := bodyMap["progress"]
		if progress != nil {
			fmt.Printf("  %s💭%s [%.0f%%] %s\n", magenta, reset, toFloat(progress)*100, truncate(content, 80))
		} else {
			fmt.Printf("  %s💭%s %s\n", magenta, reset, truncate(content, 100))
		}
	case 5: // ReasoningStep
		seq := bodyMap["sequence"]
		content, _ := bodyMap["content"].(string)
		fmt.Printf("  %s∴%s [%v] %s\n", magenta, reset, seq, truncate(content, 100))
	case 1: // Error
		code, _ := bodyMap["code"].(string)
		message, _ := bodyMap["message"].(string)
		fmt.Printf("  %s%s: %s%s\n", red, code, message, reset)
	case 35: // TitleUpdate
		title, _ := bodyMap["title"].(string)
		fmt.Printf("  %s📝%s %s\n", blue, reset, title)
	case 55: // VoiceStatus
		status, _ := bodyMap["status"].(string)
		qLen := bodyMap["queueLength"]
		fmt.Printf("  %s🎙%s  %s (queue: %v)\n", cyan, reset, status, qLen)
	case 70: // AssistantToolsRegister
		if tools, ok := bodyMap["tools"].([]interface{}); ok {
			fmt.Printf("  %s🔧%s %d tools:", yellow, reset, len(tools))
			for _, t := range tools {
				if tm, ok := t.(map[string]interface{}); ok {
					name, _ := tm["name"].(string)
					fmt.Printf(" %s%s%s", bold, name, reset)
				}
			}
			fmt.Println()
		}
	case 71: // AssistantToolsAck
		success, _ := bodyMap["success"].(bool)
		toolCount := bodyMap["toolCount"]
		if success {
			fmt.Printf("  %s✓%s %v tools registered\n", green, reset, toolCount)
		} else {
			errMsg, _ := bodyMap["error"].(string)
			fmt.Printf("  %s✗%s %s\n", red, reset, errMsg)
		}
	case 72: // AssistantHeartbeat
		// empty body — no extra output
	case 80: // GenerationComplete
		success, _ := bodyMap["success"].(bool)
		msgID, _ := bodyMap["messageId"].(string)
		if success {
			fmt.Printf("  %s✓%s complete", green, reset)
		} else {
			errMsg, _ := bodyMap["error"].(string)
			fmt.Printf("  %s✗%s failed: %s", red, reset, errMsg)
		}
		if msgID != "" {
			fmt.Printf(" %smsg:%s %s", dim, reset, msgID)
		}
		fmt.Println()
	default:
		// Generic key-value display
		printGenericBody(bodyMap)
	}
}

func printGenericBody(m map[string]interface{}) {
	for k, v := range m {
		valStr := fmt.Sprintf("%v", v)
		if len(valStr) > 100 {
			valStr = valStr[:97] + "..."
		}
		fmt.Printf("  %s%s:%s %s\n", dim, k, reset, valStr)
	}
}

func printRawMessage(num int, timestamp string, data []byte, decodeErr error) {
	fmt.Printf("%s%s#%d%s %s%s%s %s[RAW]%s (%d bytes)\n",
		dim, bgGray, num, reset,
		dim, timestamp, reset,
		red, reset,
		len(data))

	if decodeErr != nil {
		fmt.Printf("  %sdecode error: %v%s\n", dim, decodeErr, reset)
	}

	// Print hex dump (first 64 bytes)
	hexStr := hex.EncodeToString(data)
	if len(hexStr) > 128 {
		hexStr = hexStr[:128] + "..."
	}
	// Format as spaced hex pairs
	var formatted strings.Builder
	for i := 0; i < len(hexStr); i += 2 {
		if i > 0 {
			formatted.WriteByte(' ')
		}
		end := i + 2
		if end > len(hexStr) {
			end = len(hexStr)
		}
		formatted.WriteString(hexStr[i:end])
	}
	fmt.Printf("  %s%s%s\n", dim, formatted.String(), reset)
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", "↵")
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	default:
		return 0
	}
}
