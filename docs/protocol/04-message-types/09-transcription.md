### 9. Transcription (Type 9)

**Purpose:** Communicates recognized text from audio, i.e., the result of speech-to-text transcription of user's spoken input. When a user speaks, the system converts the audio to text in real-time using Whisper ASR. This message type carries those transcribed text segments (partial or final) to be treated as the user's message content.

**Audio Source:** User speech streams to the server via **LiveKit audio tracks**. The server's ASR engine (Whisper) processes the LiveKit audio track and produces Transcription messages that flow back via the data channel.

**Typical Direction:** Server → Client (the server's ASR transcribes audio from the LiveKit track and sends recognition results to the client for real-time display and conversation history)

**Fields:**

* `id` (Text, NanoID): Unique ID for this transcription message. For final transcriptions, this becomes the user message ID in the database.

* `previousId` (Text, NanoID, optional): The ID of the previous message in the conversation that this transcription follows. Typically refers to the last assistant message. May be omitted if the conversation context is sufficient.

* `conversationId` (Text): Conversation ID this transcription belongs to.

* `text` (Text): The transcript text recognized by ASR. This can be a full sentence or a partial phrase. For long utterances, multiple Transcription messages are sent as speech progresses.

* `final` (Bool, optional): Indicates if this is the final transcript for the utterance. Speech recognition systems provide interim (non-final) results that may be updated as more audio arrives. `final=true` means the ASR has finalized this segment and the user has stopped speaking (or a segment is complete). Only one final transcription per utterance is marked. If omitted, defaults to false (interim).

* `confidence` (Float, optional): Confidence score of the transcription (0.0 to 1.0). Included for informational purposes and quality monitoring.

* `language` (Text, optional): Language or locale of the recognized speech (e.g., "en-US"). Determined by ASR configuration or detection.

**MessagePack Representation (Informative):**

```
{
  "id": "trans_abc123",
  "conversationId": "conv_7H93k",
  "text": "Hello, I would like to book a flight to Paris",
  "final": true,
  "confidence": 0.92,
  "language": "en-US"
}
```

**Audio Processing Flow:**

```
User Microphone → LiveKit Audio Track → Server ASR (Whisper)
                                              ↓
                                    Transcription Messages
                                              ↓
                                    Data Channel → Client
```

**Semantics:** Transcription messages enable live transcript display and create the official user message. As the user speaks into their microphone, audio streams via a LiveKit audio track to the server. The ASR engine processes this audio stream and sends a series of Transcription messages back to the client via the data channel:

* **Example:** User says "Hello, I would like to book a flight to Paris."
* The system sends an interim transcription: `"Hello, I would like to"` with `final=false`
* Then sends another interim: `"Hello, I would like to book a flight"` with `final=false`
* Finally sends: `"Hello, I would like to book a flight to Paris"` with `final=true`

The client displays each transcription as it arrives, replacing previous interim results, until the final transcription confirms the complete utterance.

**Transcription Pipeline Semantics:**

**Interim Transcriptions (`final=false` or `final` omitted):**
* Interim transcription messages are sent as speech recognition processes the LiveKit audio track in real-time
* These are for UI display purposes only (e.g., showing live captions to provide user feedback)
* Interim transcripts are NOT stored in the conversation history
* Interim transcripts MAY be logged for debugging or audit purposes, but are not authoritative
* The client displays interim transcripts in a distinguishable way (e.g., lighter text, italic, different styling) to indicate they may change
* Each new interim transcript completely replaces the previous one in the UI
* Interim transcripts have no permanent existence; they are ephemeral UI updates

**Final Transcription (`final=true`):**
* The final Transcription message (with `final=true`) is the authoritative user message
* When the server sends a final Transcription:
  * The server creates a database entry for this user message using the transcription's `id` as the message ID
  * The client displays this as a permanent user message in the conversation UI
  * No separate UserMessage (type 2) follows from the client
  * The server immediately begins processing and responding to this message
* The final transcription's `id` field serves as the official user message ID in the database
* The `conversationId` ties this message to the conversation context
* The final transcription is treated exactly like a typed UserMessage for all processing purposes

**No Duplicate UserMessage:** The client does NOT send a separate UserMessage to echo the final transcription. The final Transcription itself constitutes the complete user message, and sending a duplicate creates ambiguity and duplicate entries. This approach reduces latency by eliminating an unnecessary round-trip and acknowledges that the server, which performs the transcription, is the authoritative source for the recognized text.

**Database Storage:**
* **Interim transcripts:** Not stored in conversation history (MAY be logged separately for audit or debugging)
* **Final transcript:** Stored as a user message with `role="user"`, using the Transcription message's `id`

**Database Alignment:**

* **Interim transcripts:** Not stored in the conversation message table. MAY be stored in a separate audit/logging table if needed, but are not part of the conversation history.
* **Final transcription:** Stored as a user message in the conversation message table (e.g., `alicia_user_messages`), with:
  * `id` = the Transcription message's `id` (NanoID)
  * `content` = the final transcription `text`
  * `role` = "user"
  * `conversationId` = from the envelope
  * Additional metadata indicates this message was voice-originated (e.g., `input_method: "voice"` or `source: "asr"`)
  * Optional: Reference to LiveKit track recording if session recording is enabled

The Transcription message type directly creates user message entries when `final=true`. Interim transcriptions do not create any conversation history entries, ensuring the database only contains the authoritative final text.

**LiveKit Integration Summary:**

* Audio Input: LiveKit audio track (user microphone → server)
* Audio Processing: Server ASR (Whisper) processes LiveKit track
* Text Output: Transcription messages via data channel (server → client)
* Message Storage: Final transcription creates database record directly
* Optional Metadata: AudioChunk (Type 4) messages MAY correlate with audio tracks for synchronization
