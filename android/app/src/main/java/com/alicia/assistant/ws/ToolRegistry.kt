package com.alicia.assistant.ws

class ToolRegistry {
    private val tools = mutableMapOf<String, ToolExecutor>()

    fun register(executor: ToolExecutor) {
        tools[executor.name] = executor
    }

    fun get(name: String): ToolExecutor? = tools[name]

    fun getAll(): List<ToolExecutor> = tools.values.toList()
}
