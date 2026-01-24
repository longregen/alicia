package com.alicia.assistant.ws

/**
 * Interface for tools that can be executed locally on the Android device.
 * Each tool is invoked by the agent via the WebSocket bridge.
 */
interface ToolExecutor {
    val name: String
    val description: String
    val inputSchema: Map<String, Any>
    suspend fun execute(arguments: Map<String, Any>): Map<String, Any>
}
