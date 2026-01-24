package com.alicia.assistant

import android.Manifest
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import android.view.View
import android.view.animation.AnimationUtils
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.viewModels
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.databinding.ActivityMainBinding
import com.alicia.assistant.model.RecognitionResult
import com.alicia.assistant.service.SaveNoteResult
import com.alicia.assistant.service.SileroVadDetector
import com.alicia.assistant.service.TtsManager
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.service.VoiceRecognitionManager
import com.alicia.assistant.service.saveRecordedNote
import com.alicia.assistant.viewmodel.MainViewModel
import com.google.android.material.color.MaterialColors
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File

class MainActivity : ComponentActivity() {
    
    private lateinit var binding: ActivityMainBinding
    private lateinit var voiceRecognitionManager: VoiceRecognitionManager
    private lateinit var ttsManager: TtsManager
    private var vadDetector: SileroVadDetector? = null
    private val vadLock = Any()
    private val viewModel: MainViewModel by viewModels()
    private var isListening = false
    private var isRecordingNote = false
    private lateinit var noteVoiceManager: VoiceRecognitionManager
    
    companion object {
        private const val REQUEST_RECORD_AUDIO = 1001
    }
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        
        voiceRecognitionManager = VoiceRecognitionManager(this, lifecycleScope)
        noteVoiceManager = VoiceRecognitionManager(this, lifecycleScope)
        ttsManager = TtsManager(this, lifecycleScope)

        setupUI()
        checkPermissions()

        lifecycleScope.launch(Dispatchers.IO) {
            val detector = SileroVadDetector.create(this@MainActivity)
            synchronized(vadLock) {
                if (vadDetector == null) {
                    vadDetector = detector
                } else {
                    detector.close()
                }
            }
        }

        if (intent?.getBooleanExtra("start_listening", false) == true) {
            intent.removeExtra("start_listening")
            lifecycleScope.launch { delay(500); startListening() }
        }
    }

    override fun onNewIntent(intent: Intent?) {
        super.onNewIntent(intent)
        if (intent?.getBooleanExtra("start_listening", false) == true) {
            intent.removeExtra("start_listening")
            lifecycleScope.launch { delay(500); startListening() }
        }
    }

    override fun onResume() {
        super.onResume()
        viewModel.refreshSettings()
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED) {
            VoiceAssistantService.ensureRunning(this)
        }
        if (!isListening && !isRecordingNote) {
            binding.statusText.text = getString(R.string.tap_to_speak)
        }
    }

    override fun onPause() {
        super.onPause()
        ttsManager.stopPlayback()
    }

    private fun setupUI() {
        binding.activationButton.setOnClickListener {
            if (isListening) {
                stopListening()
            } else {
                startListening()
            }
        }
        
        binding.noteRecordButton.setOnClickListener {
            if (isRecordingNote) {
                stopNoteRecording()
            } else {
                startNoteRecording()
            }
        }

        binding.copyTranscribedButton.setOnClickListener {
            copyToClipboard(binding.transcribedText.text)
        }

        binding.copyResponseButton.setOnClickListener {
            copyToClipboard(binding.responseText.text)
        }

        binding.conversationsButton.setOnClickListener {
            startActivity(Intent(this, ConversationListActivity::class.java))
        }

        binding.settingsButton.setOnClickListener {
            startActivity(Intent(this, SettingsActivity::class.java))
        }

        binding.notesButton.setOnClickListener {
            startActivity(Intent(this, VoiceNotesActivity::class.java))
        }

        binding.serverNotesButton.setOnClickListener {
            startActivity(Intent(this, NotesActivity::class.java))
        }
    }
    
    private fun checkPermissions() {
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO)
            != PackageManager.PERMISSION_GRANTED) {
            requestPermissions(
                arrayOf(Manifest.permission.RECORD_AUDIO),
                REQUEST_RECORD_AUDIO
            )
        }
    }
    
    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        
        when (requestCode) {
            REQUEST_RECORD_AUDIO -> {
                if (grantResults.isNotEmpty() &&
                    grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                    VoiceAssistantService.ensureRunning(this)
                } else {
                    MaterialAlertDialogBuilder(this)
                        .setTitle("Permission Required")
                        .setMessage("Microphone permission is required for voice input.")
                        .setPositiveButton("OK", null)
                        .show()
                }
            }
        }
    }
    
    private fun startListening() {
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO)
            != PackageManager.PERMISSION_GRANTED) {
            checkPermissions()
            return
        }

        isListening = true
        if (viewModel.settings.hapticFeedbackEnabled) {
            binding.activationButton.performHapticFeedback(android.view.HapticFeedbackConstants.CONFIRM)
        }
        updateUIForListening(true)

        val vad = synchronized(vadLock) {
            vadDetector ?: SileroVadDetector.create(this).also { vadDetector = it }
        }
        voiceRecognitionManager.startListeningWithVad(vad) { result ->
            lifecycleScope.launch {
                isListening = false
                updateUIForListening(false)
                binding.activationButton.isEnabled = false
                binding.statusText.text = getString(R.string.processing)

                when (result) {
                    is RecognitionResult.Success -> processVoiceInput(result.text)
                    is RecognitionResult.Error -> {
                        binding.statusText.text = getString(R.string.tap_to_speak)
                        binding.activationButton.isEnabled = true
                        Toast.makeText(this@MainActivity, "Recognition failed", Toast.LENGTH_SHORT).show()
                    }
                }
            }
        }
    }

    private fun stopListening() {
        isListening = false

        binding.activationButton.clearAnimation()
        binding.activationButton.setImageResource(R.drawable.ic_microphone)
        binding.activationButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
            MaterialColors.getColor(binding.activationButton, com.google.android.material.R.attr.colorPrimaryContainer)
        )
        binding.activationButton.isEnabled = false
        binding.waveRing1.visibility = View.GONE
        binding.waveRing2.visibility = View.GONE
        binding.statusText.text = getString(R.string.processing)
        binding.statusText.setTextColor(
            MaterialColors.getColor(binding.statusText, com.google.android.material.R.attr.colorOnSurfaceVariant)
        )

        voiceRecognitionManager.stopVadListeningEarly()
    }
    
    private fun updateUIForListening(listening: Boolean) {
        binding.statusText.text = if (listening) {
            getString(R.string.listening)
        } else {
            getString(R.string.tap_to_speak)
        }

        if (listening) {
            binding.activationButton.setImageResource(R.drawable.ic_stop)
            binding.activationButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
                getColor(R.color.recording_active)
            )
            binding.statusText.setTextColor(
                getColor(R.color.recording_active)
            )

            val pulseAnimation = AnimationUtils.loadAnimation(this, R.anim.pulse)
            binding.activationButton.startAnimation(pulseAnimation)
            binding.waveRing1.visibility = View.VISIBLE
            binding.waveRing2.visibility = View.VISIBLE
        } else {
            binding.activationButton.setImageResource(R.drawable.ic_microphone)
            binding.activationButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
                MaterialColors.getColor(binding.activationButton, com.google.android.material.R.attr.colorPrimaryContainer)
            )
            binding.statusText.setTextColor(
                MaterialColors.getColor(binding.statusText, com.google.android.material.R.attr.colorOnSurfaceVariant)
            )

            binding.activationButton.clearAnimation()
            binding.waveRing1.visibility = View.GONE
            binding.waveRing2.visibility = View.GONE
        }
    }
    
    private suspend fun processVoiceInput(text: String) {
        binding.responseCard.visibility = View.VISIBLE
        binding.transcribedText.text = text
        binding.transcribedText.visibility = View.VISIBLE
        binding.responseText.visibility = View.GONE

        val result = withContext(Dispatchers.IO) {
            viewModel.skillRouter.processInput(text)
        }

        binding.responseText.text = result.response
        binding.responseText.visibility = View.VISIBLE
        binding.statusText.text = getString(R.string.tap_to_speak)
        binding.activationButton.isEnabled = true

        viewModel.refreshSettings()
        val settings = viewModel.settings
        if (settings.voiceFeedbackEnabled) {
            ttsManager.speak(result.response, settings.ttsSpeed)
        }
    }
    
    private fun startNoteRecording() {
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO)
            != PackageManager.PERMISSION_GRANTED) {
            checkPermissions()
            return
        }
        if (isListening) return

        isRecordingNote = true
        binding.noteRecordButton.setImageResource(R.drawable.ic_stop)
        binding.noteRecordButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
            getColor(R.color.recording_active)
        )
        binding.statusText.text = getString(R.string.recording_note)
        binding.statusText.setTextColor(getColor(R.color.recording_active))
        binding.activationButton.isEnabled = false

        noteVoiceManager.startListening { result ->
            if (result is RecognitionResult.Error) {
                lifecycleScope.launch {
                    isRecordingNote = false
                    binding.noteRecordButton.setImageResource(R.drawable.ic_edit_note)
                    binding.noteRecordButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
                        MaterialColors.getColor(binding.noteRecordButton, com.google.android.material.R.attr.colorSecondaryContainer)
                    )
                    binding.statusText.text = getString(R.string.tap_to_speak)
                    binding.statusText.setTextColor(
                        MaterialColors.getColor(binding.statusText, com.google.android.material.R.attr.colorOnSurfaceVariant)
                    )
                    binding.activationButton.isEnabled = true
                    Toast.makeText(this@MainActivity, "Recording failed", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun stopNoteRecording() {
        isRecordingNote = false
        binding.noteRecordButton.setImageResource(R.drawable.ic_edit_note)
        binding.noteRecordButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
            MaterialColors.getColor(binding.noteRecordButton, com.google.android.material.R.attr.colorSecondaryContainer)
        )
        binding.statusText.setTextColor(
            MaterialColors.getColor(binding.statusText, com.google.android.material.R.attr.colorOnSurfaceVariant)
        )

        val tempFile = noteVoiceManager.stopAndGetFile()
        if (tempFile == null) {
            binding.statusText.text = getString(R.string.tap_to_speak)
            binding.activationButton.isEnabled = true
            Toast.makeText(this, "Recording failed", Toast.LENGTH_SHORT).show()
            return
        }

        binding.statusText.text = getString(R.string.tap_to_speak)
        binding.activationButton.isEnabled = true

        lifecycleScope.launch {
            val notesDir = File(filesDir, "voice_notes")
            val result = saveRecordedNote(tempFile, notesDir, noteVoiceManager, viewModel.noteRepository)
            when (result) {
                is SaveNoteResult.NoSpeechDetected ->
                    Toast.makeText(this@MainActivity, "No speech detected", Toast.LENGTH_SHORT).show()
                is SaveNoteResult.Success ->
                    Toast.makeText(this@MainActivity, getString(R.string.note_saved), Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun copyToClipboard(text: CharSequence) {
        val clipboard = getSystemService(ClipboardManager::class.java)
        clipboard.setPrimaryClip(ClipData.newPlainText("alicia", text))
        Toast.makeText(this, R.string.copied, Toast.LENGTH_SHORT).show()
    }

    override fun onDestroy() {
        super.onDestroy()
        ttsManager.destroy()
        voiceRecognitionManager.destroy()
        noteVoiceManager.destroy()
        synchronized(vadLock) {
            vadDetector?.close()
            vadDetector = null
        }
    }
}
