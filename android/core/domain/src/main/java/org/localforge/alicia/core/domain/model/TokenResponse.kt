package org.localforge.alicia.core.domain.model

/**
 * Domain model for LiveKit token response
 */
data class TokenResponse(
    val token: String,
    val expiresAt: Long,
    val roomName: String,
    val participantId: String
)
