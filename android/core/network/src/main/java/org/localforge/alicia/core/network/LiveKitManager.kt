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

@Singleton
class LiveKitManager @Inject constructor(
    private val context: Context,
    private val protocolHandler: ProtocolHandler
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private var room: Room? = null
    private var localAudioTrack: LocalAudioTrack? = null

    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Disconnected)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private var dataReceivedCallback: ((Envelope) -> Unit)? = null
    private var audioOutputEnabledCallback: (() -> Boolean)? = null

    sealed class ConnectionState {
        object Disconnected : ConnectionState()
        object Connecting : ConnectionState()
        object Connected : ConnectionState()
        object Reconnecting : ConnectionState()
        data class Failed(val error: String) : ConnectionState()
    }

    private var lastSeenStanzaId: String? = null

    private var onReconnectedCallback: (() -> Unit)? = null

    suspend fun connect(url: String, token: String) {
        try {
            _connectionState.value = ConnectionState.Connecting

            val newRoom = LiveKit.create(appContext = context)
            room = newRoom

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
                            setupAudioTracks()
                            onReconnectedCallback?.invoke()
                        }

                        is RoomEvent.Reconnecting -> {
                            Timber.i("LiveKit: Reconnecting...")
                            _connectionState.value = ConnectionState.Reconnecting
                        }

                        is RoomEvent.TrackSubscribed -> {
                            Timber.d("LiveKit: Track subscribed: ${event.track.name}")
                            if (event.track is AudioTrack) {
                                val shouldPlayAudio = audioOutputEnabledCallback?.invoke() ?: true
                                if (shouldPlayAudio) {
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
                                    envelope.stanzaId?.let { lastSeenStanzaId = it }
                                    dataReceivedCallback?.invoke(envelope)
                                } catch (e: Exception) {
                                    Timber.e(e, "Failed to decode protocol message")
                                    throw e
                                }
                            }
                        }

                        else -> {}
                    }
                }
            }

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

    private fun setupAudioTracks() {
        scope.launch {
            try {
                val audioOptions = LocalAudioTrackOptions(
                    echoCancellation = true,
                    noiseSuppression = true,
                    autoGainControl = true
                )

                room?.let { r ->
                    localAudioTrack = r.localParticipant.createAudioTrack("microphone", audioOptions)
                    localAudioTrack?.let { track ->
                        r.localParticipant.publishAudioTrack(track)
                        Timber.i("LiveKit: Published audio track")
                    }
                }
            } catch (e: Exception) {
                Timber.e(e, "Failed to setup audio tracks")
                throw e
            }
        }
    }

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
            throw e
        }
    }

    fun sendData(envelope: Envelope) {
        scope.launch {
            try {
                val data = protocolHandler.encode(envelope)
                room?.localParticipant?.publishData(
                    data,
                    io.livekit.android.room.participant.DataPublishReliability.RELIABLE
                )
                Timber.d("LiveKit: Sent data: ${envelope.type}")
            } catch (e: Exception) {
                Timber.e(e, "Failed to send data")
                throw e
            }
        }
    }

    fun onDataReceived(callback: (Envelope) -> Unit) {
        dataReceivedCallback = callback
    }

    fun onReconnected(callback: () -> Unit) {
        onReconnectedCallback = callback
    }

    fun setAudioOutputEnabledCallback(callback: () -> Boolean) {
        audioOutputEnabledCallback = callback
    }

    fun getLastSeenStanzaId(): String? {
        return lastSeenStanzaId
    }

    suspend fun setMicrophoneMuted(muted: Boolean) {
        room?.localParticipant?.let { participant ->
            participant.setMicrophoneEnabled(!muted)
            Timber.d("LiveKit: Microphone ${if (muted) "muted" else "unmuted"}")
        } ?: run {
            Timber.w("LiveKit: Cannot mute - not connected to room")
        }
    }

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

    fun isConnected(): Boolean {
        return _connectionState.value is ConnectionState.Connected
    }

    fun getCurrentRoomName(): String? {
        return room?.name
    }

    fun getCurrentParticipantId(): String? {
        return room?.localParticipant?.identity?.value
    }
}
