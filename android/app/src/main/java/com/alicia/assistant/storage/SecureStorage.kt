package com.alicia.assistant.storage

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import android.util.Log
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * Encrypts and decrypts strings using Android Keystore with AES-GCM.
 * Used for sensitive VPN credentials (auth keys).
 */
object SecureStorage {
    private const val TAG = "SecureStorage"
    private const val KEYSTORE_ALIAS = "alicia_vpn_key"
    private const val ANDROID_KEYSTORE = "AndroidKeyStore"
    private const val GCM_IV_LENGTH = 12
    private const val GCM_TAG_LENGTH = 128

    private fun getOrCreateKey(): SecretKey {
        val keyStore = KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }
        keyStore.getEntry(KEYSTORE_ALIAS, null)?.let { entry ->
            return (entry as KeyStore.SecretKeyEntry).secretKey
        }

        val keyGen = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEYSTORE)
        keyGen.init(
            KeyGenParameterSpec.Builder(
                KEYSTORE_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build()
        )
        return keyGen.generateKey()
    }

    fun encrypt(value: String): String {
        if (value.isEmpty()) return ""
        return try {
            val cipher = Cipher.getInstance("AES/GCM/NoPadding")
            cipher.init(Cipher.ENCRYPT_MODE, getOrCreateKey())
            val encrypted = cipher.doFinal(value.toByteArray(Charsets.UTF_8))
            val combined = cipher.iv + encrypted
            Base64.encodeToString(combined, Base64.NO_WRAP)
        } catch (e: Exception) {
            Log.e(TAG, "Encryption failed", e)
            ""
        }
    }

    fun decrypt(encoded: String): String {
        if (encoded.isEmpty()) return ""
        return try {
            val combined = Base64.decode(encoded, Base64.NO_WRAP)
            if (combined.size <= GCM_IV_LENGTH) return ""
            val iv = combined.copyOfRange(0, GCM_IV_LENGTH)
            val encrypted = combined.copyOfRange(GCM_IV_LENGTH, combined.size)
            val cipher = Cipher.getInstance("AES/GCM/NoPadding")
            cipher.init(Cipher.DECRYPT_MODE, getOrCreateKey(), GCMParameterSpec(GCM_TAG_LENGTH, iv))
            String(cipher.doFinal(encrypted), Charsets.UTF_8)
        } catch (e: Exception) {
            // Decryption may fail if key was rotated or value was stored unencrypted
            Log.w(TAG, "Decryption failed, returning empty", e)
            ""
        }
    }
}
