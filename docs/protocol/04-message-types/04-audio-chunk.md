### 4. AudioChunk (Type 4)

**Purpose:** Provides optional metadata for audio synchronization and correlation with LiveKit audio tracks. With LiveKit integration, actual audio streams via **LiveKit audio tracks** (not via AudioChunk messages). AudioChunk serves as metadata to correlate, synchronize, and track audio playback, recording, and analytics.

**Important:** AudioChunk does NOT transport binary audio data. Audio flows through LiveKit's real-time audio tracks. This message type exists purely for metadata and synchronization purposes.

**Typical Direction:** Bidirectional via data channel (both Client → Server and Server → Client)

**Use Cases:**

* Synchronizing text transcription or captions with audio playback timestamps
* Recording metadata for stored audio sessions (duration, format, track references)
* Debug and analytics correlation between data channel events and audio track activity
* Providing format negotiation hints or track association information

**Fields:**

* `conversationId` (Text): The conversation ID this audio metadata is associated with.
* `trackSid` (Text): The LiveKit track SID (Session ID) that this metadata references. This connects the metadata to the actual audio streaming via LiveKit.
* `format` (Text or Enum): The audio encoding format of the referenced track. For example, "audio/opus 48kHz" or "audio/pcm16 16kHz". This describes the format of the audio in the LiveKit track, not in this message.
* `duration` (UInt32, optional): Duration in milliseconds for this segment of audio. Useful for synchronization and UI progress indicators.
* `seq` (UInt32, optional): A sequence number for this metadata chunk within a stream. Helps ensure proper ordering when correlating multiple metadata messages with audio track segments. Starts at 0 or 1 and increments for each successive chunk.
* `isLast` (Bool, optional): Indicates if this is the final metadata chunk for the current utterance or audio segment. When `isLast=true`, the receiver knows no more chunks are expected for this audio stream. For user input, this signals the end of speaking. For assistant output, this marks the end of TTS audio.
* `timestamp` (UInt64, optional): Client or server timestamp (milliseconds since epoch) when this audio segment began. Enables synchronization across different event streams.

**MessagePack Representation (Informative):**

```
{
  "conversationId": "conv_7H93k",
  "trackSid": "TR_abc123xyz",
  "format": "audio/opus 48kHz",
  "duration": 320,
  "seq": 42,
  "isLast": false,
  "timestamp": 1703174400000
}
```

**Semantics:** AudioChunk messages flow through the data channel while actual audio flows through LiveKit audio tracks. These messages provide metadata that correlates with the audio:

* **User Input:** As the user speaks into their microphone, audio streams via a LiveKit audio track to the server. The client MAY send AudioChunk metadata messages to provide timing, format, or synchronization information. The server uses the LiveKit track for speech recognition, not the AudioChunk messages.

* **Assistant Output:** When the assistant responds with spoken audio (TTS), the server streams audio via a LiveKit audio track to the client. The server MAY send AudioChunk metadata to help the client synchronize captions, track progress, or log analytics.

* **Recording and Analytics:** Applications that record sessions or analyze audio quality use AudioChunk metadata to correlate events in the data channel with audio in LiveKit tracks. The `trackSid` field links metadata to specific tracks.

**Audio Transport Architecture:**

```
User Speech:
  Microphone → LiveKit Audio Track → Server ASR
                      ↓
          (Optional) AudioChunk metadata → Data Channel

Assistant Speech:
  Server TTS → LiveKit Audio Track → Client Speakers
                      ↓
          (Optional) AudioChunk metadata → Data Channel
```

**Database Alignment:** AudioChunk metadata is typically not stored in the primary conversation tables. If audio recordings are saved, they reference LiveKit track recordings or blob storage. Alicia's schema stores references or metadata for audio (such as pointers to recording files or track IDs).

AudioChunk messages do not correspond to rows in conversation history since they are transient metadata. They facilitate synchronization; the actual content of a user's speech becomes a Transcription message (Type 9), which then becomes the official user message. Therefore, AudioChunk messages do not have their own `id` field since they are not first-class conversation messages in the database. They are metadata frames that support the audio streaming infrastructure.

**Optional Nature:** Since actual audio flows via LiveKit tracks, AudioChunk messages are entirely optional. Many implementations omit them unless they need specific synchronization, analytics, or debugging capabilities. The core voice conversation works through LiveKit audio tracks and Transcription messages alone.
