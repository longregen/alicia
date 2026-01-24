package com.alicia.assistant.tools

import android.content.Context
import android.os.BatteryManager
import com.alicia.assistant.ws.ToolExecutor

class GetBatteryExecutor(private val context: Context) : ToolExecutor {
    override val name = "get_battery"
    override val description = "Get battery level and charging state of the user's phone"
    override val inputSchema = mapOf<String, Any>(
        "type" to "object",
        "properties" to emptyMap<String, Any>()
    )

    override suspend fun execute(arguments: Map<String, Any>): Map<String, Any> {
        val batteryManager = context.getSystemService(Context.BATTERY_SERVICE) as BatteryManager
        val level = batteryManager.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY)
        val charging = batteryManager.isCharging
        val status = when (batteryManager.getIntProperty(BatteryManager.BATTERY_PROPERTY_STATUS)) {
            BatteryManager.BATTERY_STATUS_CHARGING -> "charging"
            BatteryManager.BATTERY_STATUS_DISCHARGING -> "discharging"
            BatteryManager.BATTERY_STATUS_FULL -> "full"
            BatteryManager.BATTERY_STATUS_NOT_CHARGING -> "not_charging"
            else -> "unknown"
        }
        return mapOf(
            "level" to level,
            "charging" to charging,
            "status" to status,
            "unit" to "percent"
        )
    }
}
