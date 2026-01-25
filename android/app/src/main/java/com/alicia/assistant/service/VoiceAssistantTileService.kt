package com.alicia.assistant.service

import android.service.quicksettings.TileService
import com.alicia.assistant.telemetry.AliciaTelemetry
import io.opentelemetry.api.common.Attributes

class VoiceAssistantTileService : TileService() {

    override fun onClick() {
        super.onClick()
        AliciaTelemetry.withSpan("tile.toggled",
            Attributes.builder()
                .put("tile.new_state", "assist_session_triggered")
                .build()
        ) {
            AliciaInteractionService.triggerAssistSession()
        }
    }
}
