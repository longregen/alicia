package com.alicia.assistant.service

import android.content.Context
import android.net.ConnectivityManager
import java.net.NetworkInterface
import android.os.Build
import android.util.Log
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import org.json.JSONArray
import org.json.JSONObject

/**
 * Implements [libtailscale.AppContext] to provide platform services to the Go backend.
 * Handles logging, encrypted preference storage, device info, and network config.
 */
class AppContextImpl(private val context: Context) : libtailscale.AppContext {

    companion object {
        private const val TAG = "AppContextImpl"
        private const val ENCRYPTED_PREFS_FILE = "tailscale_encrypted_prefs"
    }

    private val encryptedPrefs by lazy {
        try {
            createEncryptedPrefs()
        } catch (e: Exception) {
            Log.e(TAG, "EncryptedSharedPreferences corrupted, clearing and recreating", e)
            // Delete the corrupted prefs file and try again
            try {
                context.deleteSharedPreferences(ENCRYPTED_PREFS_FILE)
            } catch (deleteErr: Exception) {
                Log.w(TAG, "Failed to delete corrupted prefs file", deleteErr)
            }
            try {
                createEncryptedPrefs()
            } catch (e2: Exception) {
                Log.e(TAG, "Failed to recreate EncryptedSharedPreferences, using fallback", e2)
                // Last resort: use regular SharedPreferences (unencrypted)
                // This is better than crashing - state will be re-established on next login
                context.getSharedPreferences(ENCRYPTED_PREFS_FILE + "_fallback", Context.MODE_PRIVATE)
            }
        }
    }

    private fun createEncryptedPrefs(): android.content.SharedPreferences {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        return EncryptedSharedPreferences.create(
            context,
            ENCRYPTED_PREFS_FILE,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }

    override fun log(tag: String, logLine: String) {
        Log.d(tag, logLine)
    }

    override fun encryptToPref(key: String, value: String) {
        try {
            encryptedPrefs.edit().putString(key, value).apply()
        } catch (e: Exception) {
            Log.e(TAG, "Failed to encrypt preference key=$key (possible keystore corruption)", e)
        }
    }

    override fun decryptFromPref(key: String): String {
        return try {
            encryptedPrefs.getString(key, "") ?: ""
        } catch (e: Exception) {
            Log.w(TAG, "Failed to decrypt preference key=$key, returning empty (possible keystore corruption)", e)
            ""
        }
    }

    override fun getStateStoreKeysJSON(): String {
        val keys = JSONArray()
        encryptedPrefs.all.keys.forEach { keys.put(it) }
        return keys.toString()
    }

    override fun getOSVersion(): String = Build.VERSION.RELEASE

    override fun getDeviceName(): String =
        "${Build.MANUFACTURER} ${Build.MODEL}".trim()

    override fun getInstallSource(): String = "alicia"

    override fun shouldUseGoogleDNSFallback(): Boolean = false

    override fun isChromeOS(): Boolean =
        context.packageManager.hasSystemFeature("android.hardware.type.pc")

    override fun getInterfacesAsJson(): String {
        Log.d(TAG, "getInterfacesAsJson called")
        val result = JSONArray()
        try {
            for (iface in NetworkInterface.getNetworkInterfaces()?.toList().orEmpty()) {
                val obj = JSONObject()
                obj.put("name", iface.name)
                obj.put("index", iface.index)
                obj.put("mtu", iface.mtu)
                obj.put("up", iface.isUp)
                obj.put("loopback", iface.isLoopback)
                obj.put("broadcast", iface.supportsMulticast())
                obj.put("pointToPoint", iface.isPointToPoint)
                obj.put("multicast", iface.supportsMulticast())
                val addrs = JSONArray()
                for (addr in iface.interfaceAddresses) {
                    val a = JSONObject()
                    a.put("ip", addr.address.hostAddress)
                    a.put("prefixLen", addr.networkPrefixLength)
                    addrs.put(a)
                }
                obj.put("addrs", addrs)
                result.put(obj)
            }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to enumerate network interfaces", e)
        }
        Log.d(TAG, "getInterfacesAsJson returning ${result.length()} interfaces")
        return result.toString()
    }

    override fun getPlatformDNSConfig(): String {
        try {
            val cm = context.getSystemService(ConnectivityManager::class.java)
            val lp = cm.getLinkProperties(cm.activeNetwork) ?: return ""
            val sb = StringBuilder()
            for (dns in lp.dnsServers) {
                if (sb.isNotEmpty()) sb.append(" ")
                sb.append(dns.hostAddress)
            }
            for (domain in lp.domains?.split(",") ?: emptyList()) {
                val trimmed = domain.trim()
                if (trimmed.isNotEmpty()) {
                    sb.append(" domain=$trimmed")
                }
            }
            return sb.toString()
        } catch (e: Exception) {
            Log.w(TAG, "Failed to get platform DNS config", e)
            return ""
        }
    }

    override fun getSyspolicyStringValue(key: String): String = throw Exception("no such key")
    override fun getSyspolicyBooleanValue(key: String): Boolean = throw Exception("no such key")
    override fun getSyspolicyStringArrayJSONValue(key: String): String = throw Exception("no such key")

    override fun hardwareAttestationKeySupported(): Boolean = false
    override fun hardwareAttestationKeyCreate(): String = ""
    override fun hardwareAttestationKeyRelease(id: String) {}
    override fun hardwareAttestationKeyPublic(id: String): ByteArray = ByteArray(0)
    override fun hardwareAttestationKeySign(id: String, data: ByteArray): ByteArray = ByteArray(0)
    override fun hardwareAttestationKeyLoad(id: String) {}
}
