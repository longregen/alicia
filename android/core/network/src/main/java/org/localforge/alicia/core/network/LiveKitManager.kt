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

    /**
     * Connection states
     */
    sealed class ConnectionState {
        object Disconnected : ConnectionState()
        object Connecting : ConnectionState()
        object Connected : ConnectionState()
        data class Failed(val error: String) : ConnectionState()
    }

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
                            Timber.e("LiveKit: Failed to connect: ${event.error.message}")
                            _connectionState.value = ConnectionState.Failed(event.error.message ?: "Connection failed")
                        }

                        is RoomEvent.Reconnected -> {
                            Timber.i("LiveKit: Reconnected to room")
                            _connectionState.value = ConnectionState.Connected
                        }

                        is RoomEvent.Reconnecting -> {
                            Timber.i("LiveKit: Reconnecting...")
                            _connectionState.value = ConnectionState.Connecting
                        }

                        is RoomEvent.TrackSubscribed -> {
                            Timber.d("LiveKit: Track subscribed: ${event.track.name}")
                            if (event.track is AudioTrack) {
                                // Start audio playback for agent's voice
                                val audioTrack = event.track as AudioTrack
                                audioTrack.start()
                                Timber.d("LiveKit: Started audio playback from ${event.participant.identity}")
                            }
                        }

                        is RoomEvent.DataReceived -> {
                            Timber.d("LiveKit: Data received: ${event.data.size} bytes from ${event.participant?.identity}")
                            scope.launch {
                                try {
                                    val envelope = protocolHandler.decode(event.data)
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
            Timber.e(e, "Failed to connect to LiveKit")
            _connectionState.value = ConnectionState.Failed(e.message ?: "Connection failed")
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
     *
     * @param envelope Protocol envelope to send
     */
    fun sendData(envelope: Envelope) {
        scope.launch {
            try {
                val data = protocolHandler.encode(envelope)
                room?.localParticipant?.publishData(data)
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
