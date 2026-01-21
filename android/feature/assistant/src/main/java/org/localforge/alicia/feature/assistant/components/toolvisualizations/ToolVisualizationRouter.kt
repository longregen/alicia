package org.localforge.alicia.feature.assistant.components.toolvisualizations

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken

@Composable
fun ToolVisualizationRouter(
    toolName: String,
    result: Any?,
    modifier: Modifier = Modifier
) {
    val gson = Gson()

    val resultMap: Map<String, Any?>? = try {
        when (result) {
            is Map<*, *> -> @Suppress("UNCHECKED_CAST") (result as Map<String, Any?>)
            is String -> {
                val type = object : TypeToken<Map<String, Any?>>() {}.type
                gson.fromJson(result, type)
            }
            else -> null
        }
    } catch (e: Exception) {
        null
    }

    when (toolName) {
        "web_read" -> WebReadVisualization(
            result = resultMap,
            modifier = modifier
        )

        "web_search" -> WebSearchVisualization(
            result = resultMap,
            modifier = modifier
        )

        "web_fetch_raw" -> WebFetchVisualization(
            result = resultMap,
            type = FetchType.RAW,
            modifier = modifier
        )

        "web_fetch_structured" -> WebFetchVisualization(
            result = resultMap,
            type = FetchType.STRUCTURED,
            modifier = modifier
        )

        "web_extract_links" -> WebLinksVisualization(
            result = resultMap,
            modifier = modifier
        )

        "web_extract_metadata" -> WebMetadataVisualization(
            result = resultMap,
            modifier = modifier
        )

        "web_screenshot" -> WebScreenshotVisualization(
            result = resultMap,
            modifier = modifier
        )

        "garden_describe_table" -> GardenTableVisualization(
            result = resultMap,
            modifier = modifier
        )

        "garden_execute_sql" -> GardenSQLVisualization(
            result = resultMap,
            modifier = modifier
        )

        "garden_schema_explore" -> GardenSchemaVisualization(
            result = resultMap,
            modifier = modifier
        )

        else -> GenericToolVisualization(
            toolName = toolName,
            result = result,
            modifier = modifier
        )
    }
}

object ToolIcons {
    val icons = mapOf(
        "web_read" to "📖",
        "web_fetch_raw" to "🌐",
        "web_fetch_structured" to "🏗️",
        "web_search" to "🔎",
        "web_extract_links" to "🔗",
        "web_extract_metadata" to "📋",
        "web_screenshot" to "📸",
        "garden_describe_table" to "📊",
        "garden_execute_sql" to "⚡",
        "garden_schema_explore" to "🗺️"
    )

    val displayNames = mapOf(
        "web_read" to "Web Read",
        "web_fetch_raw" to "Fetch Raw",
        "web_fetch_structured" to "Fetch Structured",
        "web_search" to "Web Search",
        "web_extract_links" to "Extract Links",
        "web_extract_metadata" to "Extract Metadata",
        "web_screenshot" to "Screenshot",
        "garden_describe_table" to "Describe Table",
        "garden_execute_sql" to "Execute SQL",
        "garden_schema_explore" to "Schema Explorer"
    )

    fun getIcon(toolName: String): String = icons[toolName] ?: "🔧"
    fun getDisplayName(toolName: String): String = displayNames[toolName] ?: toolName
}
