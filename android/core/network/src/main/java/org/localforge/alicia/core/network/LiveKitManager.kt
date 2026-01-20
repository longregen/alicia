package org.localforge.alicia.core.network

import android.content.Context
import org.localforge.alicia.core.network.protocol.Envelope
import org.localforge.alicia.core.network.protocol.ProtocolHandler
import timber.log.Timber
import io.livekit.android.LiveKit
import io.livekit.android.events.RoomEvent
import io.livekit.android.events.collect
import io.livekit.android.room.Room
import io.livekit.android.room.participant.RemoteParticipant
import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.LocalAudioTrack
import io.livekit.android.room.track.LocalAudioTrackOptions
import io.livekit.android.room.track.Track
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Manages LiveKit connections and handles real-time communication
 */
@Singleton
class LiveKitManager @Inject constructor(
    private val context: Context,
    private val protocolHandler: ProtocolHandler
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private var room: Room? = null
    private var localAudioTrack: LocalAudioTrack? = null

    // Connection state
    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Disconnected)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    // Callbacks
    private var dataReceivedCallback: ((Envelope) -> Unit)? = null
    private var audioOutputEnabledCallback: (() -> Boolean)? = null

    /**
     * Connection states - matches web frontend's connection state model
     */
    sealed class ConnectionState {
        object Disconnected : ConnectionState()
        object Connecting : ConnectionState()
        object Connected : ConnectionState()
        object Reconnecting : ConnectionState()
        data class Failed(val error: String) : ConnectionState()
    }

    // Track last seen stanza ID for message replay on reconnection
    private var lastSeenStanzaId: String? = null

    // Callback for when reconnection occurs (to send Configuration message)
    private var onReconnectedCallback: (() -> Unit)? = null

    /**
     * Connect to a LiveKit room
     *
     * @param url LiveKit server URL
     * @param token Authentication token
     */
    suspend fun connect(url: String, token: String) {
        try {
            _connectionState.value = ConnectionState.Connecting

            // Create room instance
            val newRoom = LiveKit.create(appContext = context)
            room = newRoom

            // Set up event collection in background
            scope.launch {
                newRoom.events.collect { event ->
                    when (event) {
                        is RoomEvent.Connected -> {
                            Timber.i("LiveKit: Connected to room")
                            _connectionState.value = ConnectionState.Connected
                        }

                        is RoomEvent.Disconnected -> {
                            val error = event.error
                            Timber.i("LiveKit: Disconnected from room. Error: ${error?.message}")
                            _connectionState.value = if (error != null) {
                                ConnectionState.Failed(error.message ?: "Unknown error")
                            } else {
                                ConnectionState.Disconnected
                            }
                        }

                        is RoomEvent.FailedToConnect -> {
                            val errorMessage = mapConnectionError(event.error)
                            Timber.e("LiveKit: Failed to connect: ${event.error.message}")
                            _connectionState.value = ConnectionState.Failed(errorMessage)
                        }

                        is RoomEvent.Reconnected -> {
                            Timber.i("LiveKit: Reconnected to room")
                            _connectionState.value = ConnectionState.Connected
                            // Re-publish audio tracks after reconnection (matching web behavior)
                            setupAudioTracks()
                            // Notify callback to send Configuration message with lastSeenStanzaId
                            onReconnectedCallback?.invoke()
                        }

                        is RoomEvent.Reconnecting -> {
                            Timber.i("LiveKit: Reconnecting...")
                            _connectionState.value = ConnectionState.Reconnecting
                        }

                        is RoomEvent.TrackSubscribed -> {
                            Timber.d("LiveKit: Track subscribed: ${event.track.name}")
                            if (event.track is AudioTrack) {
                                // Check if audio output is enabled before playing
                                val shouldPlayAudio = audioOutputEnabledCallback?.invoke() ?: true
                                if (shouldPlayAudio) {
                                    // Start audio playback for agent's voice
                                    val audioTrack = event.track as AudioTrack
                                    audioTrack.start()
                                    Timber.d("LiveKit: Started audio playback from ${event.participant.identity}")
                                } else {
                                    Timber.d("LiveKit: Audio output disabled, skipping playback from ${event.participant.identity}")
                                }
                            }
                        }

                        is RoomEvent.DataReceived -> {
                            Timber.d("LiveKit: Data received: ${event.data.size} bytes from ${event.participant?.identity}")
                            scope.launch {
                                try {
                                    val envelope = protocolHandler.decode(event.data)
                                    // Track last seen stanza ID for message replay on reconnection
                                    envelope.stanzaId?.let { lastSeenStanzaId = it }
                                    dataReceivedCallback?.invoke(envelope)
                                } catch (e: Exception) {
                                    Timber.e(e, "Failed to decode protocol message")
                                }
                            }
                        }

                        else -> {
                            // Ignore other events
                        }
                    }
                }
            }

            // Connect to the room - this suspends until connection is established
            newRoom.connect(url, token)

            Timber.i("LiveKit: Connected to room ${room?.name}")
            setupAudioTracks()

        } catch (e: Exception) {
            val errorMessage = mapConnectionError(e)
            Timber.e(e, "Failed to connect to LiveKit: $errorMessage")
            _connectionState.value = ConnectionState.Failed(errorMessage)
            throw e
        }
    }

    /**
     * Set up local audio tracks for publishing
     */
    private fun setupAudioTracks() {
        scope.launch {
            try {
                val audioOptions = LocalAudioTrackOptions(
                    echoCancellation = true,
                    noiseSuppression = true,
                    autoGainControl = true
                )

                // Create and publish local audio track
                room?.let { r ->
                    localAudioTrack = r.localParticipant.createAudioTrack("microphone", audioOptions)
                    localAudioTrack?.let { track ->
                        r.localParticipant.publishAudioTrack(track)
                        Timber.i("LiveKit: Published audio track")
                    }
                }
            } catch (e: Exception) {
                Timber.e(e, "Failed to setup audio tracks")
            }
        }
    }

    /**
     * Disconnect from the current room
     */
    fun disconnect() {
        try {
            localAudioTrack?.stop()
            localAudioTrack = null

            room?.disconnect()
            room = null

            _connectionState.value = ConnectionState.Disconnected
            Timber.i("LiveKit: Disconnected")
        } catch (e: Exception) {
            Timber.e(e, "Error during disconnect")
        }
    }

    /**
     * Send a protocol message to the room
     * Uses reliable data channel (matching web's reliable: true)
     *
     * @param envelope Protocol envelope to send
     */
    fun sendData(envelope: Envelope) {
        scope.launch {
            try {
                val data = protocolHandler.encode(envelope)
                // Use reliable data channel (matching web's { reliable: true })
                room?.localParticipant?.publishData(
                    data,
                    io.livekit.android.room.participant.DataPublishReliability.RELIABLE
                )
                Timber.d("LiveKit: Sent data: ${envelope.type}")
            } catch (e: Exception) {
                Timber.e(e, "Failed to send data")
            }
        }
    }

    /**
     * Set callback for when data messages are received
     *
     * @param callback Function to invoke when a protocol envelope is received via LiveKit data channel
     */
    fun onDataReceived(callback: (Envelope) -> Unit) {
        dataReceivedCallback = callback
    }

    /**
     * Set callback for when reconnection occurs.
     * Caller should send a Configuration message with lastSeenStanzaId to trigger message replay.
     *
     * @param callback Function to invoke after successful reconnection
     */
    fun onReconnected(callback: () -> Unit) {
        onReconnectedCallback = callback
    }

    /**
     * Set callback to check if audio output is enabled.
     * Called before playing remote audio tracks. If not set, defaults to true (always play).
     *
     * @param callback Function that returns true if audio output should be played
     */
    fun setAudioOutputEnabledCallback(callback: () -> Boolean) {
        audioOutputEnabledCallback = callback
    }

    /**
     * Get the last seen stanza ID for message replay on reconnection.
     * This should be sent in the Configuration message after reconnection.
     *
     * @return The last seen stanza ID, or null if no messages received yet
     */
    fun getLastSeenStanzaId(): String? {
        return lastSeenStanzaId
    }

    /**
     * Mute or unmute the local microphone.
     *
     * When muted, audio data stops being transmitted to the server and remote
     * participants are notified via TrackMutedEvent. The track remains active
     * and can be unmuted without reinitialization.
     *
     * @param muted true to mute (stops transmission), false to unmute (resumes transmission)
     */
    suspend fun setMicrophoneMuted(muted: Boolean) {
        room?.localParticipant?.let { participant ->
            participant.setMicrophoneEnabled(!muted)
            Timber.d("LiveKit: Microphone ${if (muted) "muted" else "unmuted"}")
        } ?: run {
            Timber.w("LiveKit: Cannot mute - not connected to room")
        }
    }

    /**
     * Map connection errors to user-friendly messages.
     * Matches web frontend's error message patterns for consistency.
     */
    private fun mapConnectionError(error: Throwable): String {
        val message = error.message ?: "Unknown error"
        return when {
            message.contains("Network", ignoreCase = true) ||
            message.contains("UnknownHostException", ignoreCase = true) ||
            message.contains("ConnectException", ignoreCase = true) ->
                "Network error: Unable to reach the voice service. Please check your connection."

            message.contains("token", ignoreCase = true) ||
            message.contains("auth", ignoreCase = true) ||
            message.contains("401") ||
            message.contains("403") ->
                "Authentication failed. Please try creating a new conversation."

            message.contains("timeout", ignoreCase = true) ||
            message.contains("SocketTimeoutException", ignoreCase = true) ->
                "Connection timeout. The voice service may be unavailable."

            else -> message
        }
    }

    /**
     * Check if currently connected
     *
     * @return true if connection state is Connected, false for Disconnected/Connecting/Failed states
     */
    fun isConnected(): Boolean {
        return _connectionState.value is ConnectionState.Connected
    }

    /**
     * Get current room name
     *
     * @return The name of the currently connected room, or null if not connected
     */
    fun getCurrentRoomName(): String? {
        return room?.name
    }

    /**
     * Get current participant ID
     *
     * @return The identity of the local participant, or null if not connected
     */
    fun getCurrentParticipantId(): String? {
        return room?.localParticipant?.identity?.value
    }
}
