package com.alicia.assistant.service

import android.service.quicksettings.TileService

class VoiceAssistantTileService : TileService() {

    override fun onClick() {
        super.onClick()
        AliciaInteractionService.triggerAssistSession()
    }
}
