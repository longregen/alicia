package com.alicia.assistant

import android.Manifest
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import android.util.Log
import android.animation.ArgbEvaluator
import android.animation.ValueAnimator
import android.content.res.Configuration
import android.view.View
import android.view.animation.AnimationUtils
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.viewModels
import androidx.lifecycle.lifecycleScope
import com.alicia.assistant.databinding.ActivityMainBinding
import com.alicia.assistant.model.RecognitionResult
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.service.SaveNoteResult
import com.alicia.assistant.service.SileroVadDetector
import com.alicia.assistant.service.TtsManager
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.service.VoiceRecognitionManager
import com.alicia.assistant.service.saveRecordedNote
import com.alicia.assistant.storage.PreferencesManager
import com.alicia.assistant.viewmodel.MainViewModel
import com.google.android.material.color.MaterialColors
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class MainActivity : ComponentActivity() {

    companion object {
        /** Allow UI to settle before activating microphone */
        private const val UI_SETTLE_DELAY_MS = 500L
    }

    private lateinit var binding: ActivityMainBinding
    private lateinit var voiceRecognitionManager: VoiceRecognitionManager
    private lateinit var ttsManager: TtsManager
    private var isSetupComplete = false
    private var vadDetector: SileroVadDetector? = null
    private val vadLock = Any()
    private val viewModel: MainViewModel by viewModels()
    private var isListening = false
    private var isRecordingNote = false
    private lateinit var noteVoiceManager: VoiceRecognitionManager
    private val apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
    private var backgroundAnimator: ValueAnimator? = null
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (savedInstanceState == null) {
            lifecycleScope.launch {
                if (!PreferencesManager(this@MainActivity).isOnboardingCompleted()) {
                    startActivity(Intent(this@MainActivity, OnboardingActivity::class.java))
                    finish()
                    return@launch
                }
                continueWithSetup()
            }
            return
        }

        continueWithSetup()
    }

    private fun continueWithSetup() {
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        voiceRecognitionManager = VoiceRecognitionManager(this, lifecycleScope)
        noteVoiceManager = VoiceRecognitionManager(this, lifecycleScope)
        ttsManager = TtsManager(this, lifecycleScope)

        setupUI()

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
            lifecycleScope.launch {
                // Allow UI to settle before activating microphone
                delay(UI_SETTLE_DELAY_MS)
                startListening()
            }
        }

        isSetupComplete = true
    }

    override fun onNewIntent(intent: Intent?) {
        super.onNewIntent(intent)
        if (intent?.getBooleanExtra("start_listening", false) == true) {
            intent.removeExtra("start_listening")
            lifecycleScope.launch {
                // Allow UI to settle before activating microphone
                delay(UI_SETTLE_DELAY_MS)
                startListening()
            }
        }
    }

    override fun onResume() {
        super.onResume()
        if (!isSetupComplete) return

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
        if (!isSetupComplete) return
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
    }

    private fun requireRecordPermission(): Boolean {
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO)
            == PackageManager.PERMISSION_GRANTED) {
            return true
        }
        Toast.makeText(this, R.string.microphone_permission_required, Toast.LENGTH_SHORT).show()
        return false
    }

    private fun startListening() {
        if (!requireRecordPermission()) return
        if (isRecordingNote) return

        isListening = true
        if (viewModel.settings.value.hapticFeedbackEnabled) {
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
                        Toast.makeText(this@MainActivity, R.string.recognition_failed, Toast.LENGTH_SHORT).show()
                    }
                }
            }
        }
    }

    private fun stopListening() {
        isListening = false

        // Stop all animations
        binding.activationButton.clearAnimation()
        binding.waveRing1.clearAnimation()
        binding.waveRing2.clearAnimation()
        binding.waveRing3.clearAnimation()
        backgroundAnimator?.cancel()

        binding.activationButton.setImageResource(R.drawable.ic_microphone)
        binding.activationButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
            MaterialColors.getColor(binding.activationButton, com.google.android.material.R.attr.colorPrimaryContainer)
        )
        binding.activationButton.isEnabled = false
        binding.waveRing1.visibility = View.GONE
        binding.waveRing2.visibility = View.GONE
        binding.waveRing3.visibility = View.GONE

        // Reset background
        binding.listeningOverlay.setBackgroundColor(
            MaterialColors.getColor(binding.listeningOverlay, com.google.android.material.R.attr.colorSurface)
        )

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
            // Update button appearance
            binding.activationButton.setImageResource(R.drawable.ic_stop)
            binding.activationButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
                getColor(R.color.recording_active)
            )
            binding.activationButton.imageTintList = android.content.res.ColorStateList.valueOf(
                android.graphics.Color.WHITE
            )
            binding.statusText.setTextColor(
                getColor(R.color.recording_active)
            )

            // Start button pulse animation
            val pulseAnimation = AnimationUtils.loadAnimation(this, R.anim.pulse)
            binding.activationButton.startAnimation(pulseAnimation)

            // Show and animate wave rings with staggered expanding animations
            binding.waveRing1.visibility = View.VISIBLE
            binding.waveRing2.visibility = View.VISIBLE
            binding.waveRing3.visibility = View.VISIBLE

            val waveAnim1 = AnimationUtils.loadAnimation(this, R.anim.wave_expand_1)
            val waveAnim2 = AnimationUtils.loadAnimation(this, R.anim.wave_expand_2)
            val waveAnim3 = AnimationUtils.loadAnimation(this, R.anim.wave_expand_3)

            binding.waveRing1.startAnimation(waveAnim1)
            binding.waveRing2.startAnimation(waveAnim2)
            binding.waveRing3.startAnimation(waveAnim3)

            // Animate background to subtle red tint
            val surfaceColor = MaterialColors.getColor(
                binding.listeningOverlay,
                com.google.android.material.R.attr.colorSurface
            )
            val isDarkMode = (resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK) ==
                Configuration.UI_MODE_NIGHT_YES
            val tintColor = if (isDarkMode) {
                getColor(R.color.recording_background_tint_dark)
            } else {
                getColor(R.color.recording_background_tint_light)
            }

            backgroundAnimator?.cancel()
            backgroundAnimator = ValueAnimator.ofObject(ArgbEvaluator(), surfaceColor, tintColor).apply {
                duration = 400
                addUpdateListener { animator ->
                    binding.listeningOverlay.setBackgroundColor(animator.animatedValue as Int)
                }
                start()
            }
        } else {
            // Update button appearance
            binding.activationButton.setImageResource(R.drawable.ic_microphone)
            binding.activationButton.backgroundTintList = android.content.res.ColorStateList.valueOf(
                MaterialColors.getColor(binding.activationButton, com.google.android.material.R.attr.colorPrimaryContainer)
            )
            binding.activationButton.imageTintList = android.content.res.ColorStateList.valueOf(
                MaterialColors.getColor(binding.activationButton, com.google.android.material.R.attr.colorOnPrimaryContainer)
            )
            binding.statusText.setTextColor(
                MaterialColors.getColor(binding.statusText, com.google.android.material.R.attr.colorOnSurfaceVariant)
            )

            // Stop all animations
            binding.activationButton.clearAnimation()
            binding.waveRing1.clearAnimation()
            binding.waveRing2.clearAnimation()
            binding.waveRing3.clearAnimation()
            backgroundAnimator?.cancel()

            // Hide wave rings
            binding.waveRing1.visibility = View.GONE
            binding.waveRing2.visibility = View.GONE
            binding.waveRing3.visibility = View.GONE

            // Reset background color
            val surfaceColor = MaterialColors.getColor(
                binding.listeningOverlay,
                com.google.android.material.R.attr.colorSurface
            )
            binding.listeningOverlay.setBackgroundColor(surfaceColor)
        }
    }

    private suspend fun processVoiceInput(text: String) {
        binding.responseCard.visibility = View.VISIBLE
        binding.transcribedText.text = text
        binding.transcribedText.visibility = View.VISIBLE
        binding.responseText.visibility = View.GONE

        val response = try {
            withContext(Dispatchers.IO) {
                val dateFormat = SimpleDateFormat("MMM d, h:mm a", Locale.getDefault())
                val title = "Voice ${dateFormat.format(Date())}"
                val conversation = apiClient.createConversation(title)
                apiClient.sendMessageSync(conversation.id, text).assistantMessage.content
            }
        } catch (e: Exception) {
            Log.e("MainActivity", "Voice interaction failed", e)
            "Sorry, I couldn't get a response right now."
        }

        binding.responseText.text = response
        binding.responseText.visibility = View.VISIBLE
        binding.statusText.text = getString(R.string.tap_to_speak)
        binding.activationButton.isEnabled = true

        viewModel.refreshSettings()
        val settings = viewModel.settings.value
        if (settings.voiceFeedbackEnabled) {
            ttsManager.speak(response, settings.ttsSpeed)
        }
    }
    
    private fun startNoteRecording() {
        if (!requireRecordPermission()) return
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
                    Toast.makeText(this@MainActivity, R.string.recording_failed, Toast.LENGTH_SHORT).show()
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
            Toast.makeText(this, R.string.recording_failed, Toast.LENGTH_SHORT).show()
            return
        }

        binding.statusText.text = getString(R.string.tap_to_speak)
        binding.activationButton.isEnabled = true

        lifecycleScope.launch {
            val notesDir = File(filesDir, "voice_notes")
            val result = saveRecordedNote(tempFile, notesDir, noteVoiceManager, viewModel.noteRepository, apiClient)
            when (result) {
                is SaveNoteResult.NoSpeechDetected ->
                    Toast.makeText(this@MainActivity, R.string.no_speech_detected, Toast.LENGTH_SHORT).show()
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
        if (!isSetupComplete) return

        backgroundAnimator?.cancel()
        backgroundAnimator = null
        ttsManager.destroy()
        voiceRecognitionManager.destroy()
        noteVoiceManager.destroy()
        synchronized(vadLock) {
            vadDetector?.close()
            vadDetector = null
        }
    }
}
