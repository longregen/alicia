package com.alicia.assistant

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import android.util.Log
import android.widget.EditText
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.ImageProxy
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.service.VpnManager
import com.alicia.assistant.storage.PreferencesManager
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.mlkit.vision.barcode.BarcodeScanning
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.common.InputImage
import kotlinx.coroutines.launch
import java.util.concurrent.Executors

@androidx.camera.core.ExperimentalGetImage
class VpnQrScanActivity : ComponentActivity() {

    companion object {
        private const val TAG = "VpnQrScan"
        private const val CAMERA_PERMISSION_REQUEST = 1001
    }

    @Volatile
    private var scanComplete = false
    private val cameraExecutor = Executors.newSingleThreadExecutor()
    private val barcodeScanner = BarcodeScanning.getClient()
    private lateinit var preferencesManager: PreferencesManager

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_vpn_qr_scan)
        preferencesManager = PreferencesManager(this)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        toolbar.setNavigationOnClickListener { finish() }

        findViewById<com.google.android.material.button.MaterialButton>(R.id.enterManuallyButton)
            .setOnClickListener { showManualEntryDialog() }

        if (checkSelfPermission(Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED) {
            startCamera()
        } else {
            requestPermissions(arrayOf(Manifest.permission.CAMERA), CAMERA_PERMISSION_REQUEST)
        }
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == CAMERA_PERMISSION_REQUEST) {
            if (grantResults.isNotEmpty() && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                startCamera()
            } else {
                Toast.makeText(this, getString(R.string.vpn_camera_permission_required), Toast.LENGTH_LONG).show()
                finish()
            }
        }
    }

    private fun startCamera() {
        val cameraProviderFuture = ProcessCameraProvider.getInstance(this)
        cameraProviderFuture.addListener({
            val cameraProvider = cameraProviderFuture.get()
            val preview = Preview.Builder().build()
            val previewView = findViewById<PreviewView>(R.id.cameraPreview)
            preview.surfaceProvider = previewView.surfaceProvider

            val imageAnalysis = ImageAnalysis.Builder()
                .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                .build()

            imageAnalysis.setAnalyzer(cameraExecutor) { imageProxy ->
                processImage(imageProxy)
            }

            try {
                cameraProvider.unbindAll()
                cameraProvider.bindToLifecycle(
                    this,
                    CameraSelector.DEFAULT_BACK_CAMERA,
                    preview,
                    imageAnalysis
                )
            } catch (e: Exception) {
                Log.e(TAG, "Camera binding failed", e)
            }
        }, ContextCompat.getMainExecutor(this))
    }

    private fun processImage(imageProxy: ImageProxy) {
        if (scanComplete) {
            imageProxy.close()
            return
        }

        val mediaImage = imageProxy.image
        if (mediaImage == null) {
            imageProxy.close()
            return
        }

        val inputImage = InputImage.fromMediaImage(mediaImage, imageProxy.imageInfo.rotationDegrees)

        barcodeScanner.process(inputImage)
            .addOnSuccessListener { barcodes ->
                for (barcode in barcodes) {
                    if (barcode.valueType == Barcode.TYPE_TEXT || barcode.valueType == Barcode.TYPE_URL) {
                        val value = barcode.rawValue ?: continue
                        if (!scanComplete) {
                            scanComplete = true
                            handleScannedCode(value)
                        }
                        break
                    }
                }
            }
            .addOnCompleteListener {
                imageProxy.close()
            }
    }

    private fun handleScannedCode(code: String) {
        Log.i(TAG, "QR code scanned: ${code.take(20)}...")
        lifecycleScope.launch {
            val (authKey, serverUrl) = parseQrCode(code)

            if (serverUrl != null) {
                VpnManager.loginWithUrl(serverUrl)
                preferencesManager.saveVpnSettings(
                    preferencesManager.getVpnSettings().copy(
                        headscaleUrl = serverUrl,
                        authKey = authKey,
                        nodeRegistered = true
                    )
                )
            } else {
                preferencesManager.saveVpnSettings(
                    preferencesManager.getVpnSettings().copy(
                        authKey = authKey,
                        nodeRegistered = true
                    )
                )
            }

            VpnManager.loginWithAuthKey(authKey)

            Toast.makeText(this@VpnQrScanActivity, getString(R.string.vpn_credentials_saved), Toast.LENGTH_SHORT).show()
            finish()
        }
    }

    private fun parseQrCode(code: String): Pair<String, String?> {
        return try {
            if (code.startsWith("headscale://auth")) {
                val uri = android.net.Uri.parse(code)
                val key = uri.getQueryParameter("key") ?: code
                val server = uri.getQueryParameter("server")
                Pair(key, server)
            } else {
                Pair(code, null)
            }
        } catch (e: Exception) {
            Pair(code, null)
        }
    }

    private fun showManualEntryDialog() {
        val input = EditText(this).apply {
            hint = getString(R.string.vpn_pre_auth_key_hint)
            setPadding(48, 32, 48, 16)
        }

        MaterialAlertDialogBuilder(this)
            .setTitle(getString(R.string.vpn_enter_auth_key))
            .setView(input)
            .setPositiveButton(getString(R.string.vpn_submit)) { _, _ ->
                val key = input.text.toString().trim()
                if (key.isNotEmpty()) {
                    handleScannedCode(key)
                }
            }
            .setNegativeButton(getString(R.string.vpn_cancel), null)
            .show()
    }

    override fun onDestroy() {
        super.onDestroy()
        cameraExecutor.shutdown()
        barcodeScanner.close()
    }
}
