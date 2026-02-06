package com.alicia.assistant

import android.Manifest
import android.app.role.RoleManager
import android.content.Intent
import android.os.Bundle
import android.provider.Settings
import android.view.View
import android.widget.LinearLayout
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts
import androidx.lifecycle.lifecycleScope
import androidx.viewpager2.widget.ViewPager2
import com.alicia.assistant.databinding.ActivityOnboardingBinding
import com.alicia.assistant.storage.PreferencesManager
import kotlinx.coroutines.launch

class OnboardingActivity : ComponentActivity() {

    private lateinit var binding: ActivityOnboardingBinding
    private lateinit var pagerAdapter: OnboardingPagerAdapter
    private val indicators = mutableListOf<View>()
    private val roleManager: RoleManager? by lazy { getSystemService(RoleManager::class.java) }

    private val micPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        handlePermissionResult(OnboardingPage.MICROPHONE, granted)
    }

    private val notificationPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        handlePermissionResult(OnboardingPage.NOTIFICATIONS, granted)
    }

    private val bluetoothPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        handlePermissionResult(OnboardingPage.BLUETOOTH, granted)
    }

    private val locationPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        handlePermissionResult(OnboardingPage.LOCATION, granted)
    }

    private val assistantRoleLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) {
        pagerAdapter.refreshPages()
        rebuildPageIndicators()
        if (roleManager?.isRoleHeld(RoleManager.ROLE_ASSISTANT) == true) {
            Toast.makeText(this, R.string.assistant_configured, Toast.LENGTH_SHORT).show()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityOnboardingBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setupViewPager()
        setupButtons()
        setupPageIndicators()
    }

    private fun setupViewPager() {
        pagerAdapter = OnboardingPagerAdapter(
            onGrantPermission = { page -> requestPermission(page) },
            getPermissionStatus = { page -> isPermissionGranted(page) },
            onSetupAssistant = { requestAssistantRole() },
            isAssistantConfigured = { roleManager?.isRoleHeld(RoleManager.ROLE_ASSISTANT) == true }
        )
        binding.viewPager.adapter = pagerAdapter
        binding.viewPager.isUserInputEnabled = false

        binding.viewPager.registerOnPageChangeCallback(object : ViewPager2.OnPageChangeCallback() {
            override fun onPageSelected(position: Int) {
                updatePageIndicators(position)
                updateButtons(position)
                refreshCurrentPage()
            }
        })
    }

    private fun setupButtons() {
        binding.nextButton.setOnClickListener {
            val currentPage = pagerAdapter.getPage(binding.viewPager.currentItem)

            if (currentPage.isPermissionPage && !currentPage.isOptional && !isPermissionGranted(currentPage)) {
                Toast.makeText(this, R.string.permission_required_to_continue, Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            if (binding.viewPager.currentItem < pagerAdapter.itemCount - 1) {
                binding.viewPager.currentItem++
            } else {
                completeOnboarding()
            }
        }

        binding.skipButton.setOnClickListener {
            if (binding.viewPager.currentItem < pagerAdapter.itemCount - 1) {
                binding.viewPager.currentItem++
            }
        }

        updateButtons(0)
    }

    private fun setupPageIndicators() {
        val params = LinearLayout.LayoutParams(
            resources.getDimensionPixelSize(R.dimen.indicator_size),
            resources.getDimensionPixelSize(R.dimen.indicator_size)
        ).apply {
            marginStart = resources.getDimensionPixelSize(R.dimen.indicator_margin)
            marginEnd = resources.getDimensionPixelSize(R.dimen.indicator_margin)
        }

        repeat(pagerAdapter.itemCount) {
            val indicator = View(this).apply {
                layoutParams = params
                setBackgroundResource(R.drawable.indicator_inactive)
            }
            indicators.add(indicator)
            binding.indicatorContainer.addView(indicator)
        }

        updatePageIndicators(0)
    }

    private fun updatePageIndicators(position: Int) {
        indicators.forEachIndexed { index, view ->
            view.setBackgroundResource(
                if (index == position) R.drawable.indicator_active else R.drawable.indicator_inactive
            )
        }
    }

    private fun updateButtons(position: Int) {
        val page = pagerAdapter.getPage(position)
        val isLastPage = position == pagerAdapter.itemCount - 1

        binding.nextButton.text = getString(if (isLastPage) R.string.get_started else R.string.next)
        binding.skipButton.visibility = if (page.isOptional && !isLastPage) View.VISIBLE else View.GONE
    }

    private fun requestPermission(page: OnboardingPage) {
        when (page) {
            OnboardingPage.MICROPHONE -> micPermissionLauncher.launch(Manifest.permission.RECORD_AUDIO)
            OnboardingPage.NOTIFICATIONS -> notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            OnboardingPage.BLUETOOTH -> bluetoothPermissionLauncher.launch(Manifest.permission.BLUETOOTH_CONNECT)
            OnboardingPage.LOCATION -> locationPermissionLauncher.launch(Manifest.permission.ACCESS_COARSE_LOCATION)
            else -> {}
        }
    }

    private fun requestAssistantRole() {
        // RoleManager.createRequestRoleIntent(ROLE_ASSISTANT) is silently rejected
        // on Google-branded images where the role is marked non-requestable.
        // Open the system voice-input settings instead, which works everywhere.
        assistantRoleLauncher.launch(Intent(Settings.ACTION_VOICE_INPUT_SETTINGS))
    }

    private fun handlePermissionResult(page: OnboardingPage, granted: Boolean) {
        pagerAdapter.refreshPages()
        rebuildPageIndicators()
        if (granted) {
            Toast.makeText(this, R.string.permission_granted, Toast.LENGTH_SHORT).show()
        } else if (!page.isOptional) {
            Toast.makeText(this, R.string.permission_denied, Toast.LENGTH_SHORT).show()
        }
    }

    private fun refreshCurrentPage() {
        pagerAdapter.notifyItemChanged(binding.viewPager.currentItem)
    }

    private fun rebuildPageIndicators() {
        binding.indicatorContainer.removeAllViews()
        indicators.clear()
        setupPageIndicators()
        updateButtons(binding.viewPager.currentItem)
    }

    private fun isPermissionGranted(page: OnboardingPage): Boolean {
        val permission = when (page) {
            OnboardingPage.MICROPHONE -> Manifest.permission.RECORD_AUDIO
            OnboardingPage.NOTIFICATIONS -> Manifest.permission.POST_NOTIFICATIONS
            OnboardingPage.BLUETOOTH -> Manifest.permission.BLUETOOTH_CONNECT
            OnboardingPage.LOCATION -> Manifest.permission.ACCESS_COARSE_LOCATION
            else -> return true
        }
        return checkSelfPermission(permission) == android.content.pm.PackageManager.PERMISSION_GRANTED
    }

    private fun completeOnboarding() {
        lifecycleScope.launch {
            PreferencesManager(this@OnboardingActivity).setOnboardingCompleted(true)
            startActivity(Intent(this@OnboardingActivity, MainActivity::class.java))
            finish()
        }
    }
}
