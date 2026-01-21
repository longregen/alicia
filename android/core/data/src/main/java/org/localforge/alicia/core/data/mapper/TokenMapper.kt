package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.network.model.TokenResponse as NetworkTokenResponse
import org.localforge.alicia.core.domain.model.TokenResponse as DomainTokenResponse

fun NetworkTokenResponse.toDomain(): DomainTokenResponse {
    return DomainTokenResponse(
        token = this.token,
        expiresAt = this.expiresAt,
        roomName = this.roomName,
        participantId = this.participantId
    )
}

fun DomainTokenResponse.toNetwork(): NetworkTokenResponse {
    return NetworkTokenResponse(
        token = this.token,
        expiresAt = this.expiresAt,
        roomName = this.roomName,
        participantId = this.participantId
    )
}
