package org.localforge.alicia.core.domain.model

data class TokenResponse(
    val token: String,
    val expiresAt: Long,
    val roomName: String,
    val participantId: String
)
