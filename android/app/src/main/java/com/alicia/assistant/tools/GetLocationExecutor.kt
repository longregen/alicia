package com.alicia.assistant.tools

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Geocoder
import android.location.LocationManager
import androidx.core.content.ContextCompat
import com.alicia.assistant.ws.ToolExecutor
import java.util.Locale

class GetLocationExecutor(private val context: Context) : ToolExecutor {
    override val name = "get_location"
    override val description = "Get the user's current location (city-level coarse location) from their phone"
    override val inputSchema = mapOf<String, Any>(
        "type" to "object",
        "properties" to emptyMap<String, Any>()
    )

    override suspend fun execute(arguments: Map<String, Any>): Map<String, Any> {
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION)
            != PackageManager.PERMISSION_GRANTED
        ) {
            return mapOf("error" to "Location permission not granted")
        }

        val locationManager = context.getSystemService(Context.LOCATION_SERVICE) as LocationManager
        val location = locationManager.getLastKnownLocation(LocationManager.NETWORK_PROVIDER)
            ?: locationManager.getLastKnownLocation(LocationManager.GPS_PROVIDER)
            ?: return mapOf("error" to "No location available")

        val result = mutableMapOf<String, Any>(
            "latitude" to location.latitude,
            "longitude" to location.longitude,
            "accuracy" to location.accuracy.toDouble()
        )

        // Reverse geocode for city name
        try {
            val geocoder = Geocoder(context, Locale.getDefault())
            @Suppress("DEPRECATION")
            val addresses = geocoder.getFromLocation(location.latitude, location.longitude, 1)
            if (!addresses.isNullOrEmpty()) {
                val addr = addresses[0]
                addr.locality?.let { result["city"] = it }
                addr.adminArea?.let { result["region"] = it }
                addr.countryName?.let { result["country"] = it }
            }
        } catch (_: Exception) {
            // Geocoding is best-effort
        }

        return result
    }
}
