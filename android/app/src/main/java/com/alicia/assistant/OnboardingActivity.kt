package com.alicia.assistant

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
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
    private lateinit var preferencesManager: PreferencesManager
    private lateinit var pagerAdapter: OnboardingPagerAdapter
    private var pendingPermissionPage: OnboardingPage? = null
    private val indicators = mutableListOf<View>()

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

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityOnboardingBinding.inflate(layoutInflater)
        setContentView(binding.root)

        preferencesManager = PreferencesManager(this)
        setupViewPager()
        setupButtons()
        setupPageIndicators()
    }

    private fun setupViewPager() {
        pagerAdapter = OnboardingPagerAdapter(
            onGrantPermission = { page -> requestPermissionForPage(page) },
            getPermissionStatus = { page -> isPermissionGranted(page) }
        )
        binding.viewPager.adapter = pagerAdapter
        binding.viewPager.isUserInputEnabled = false // Disable swipe, use buttons

        binding.viewPager.registerOnPageChangeCallback(object : ViewPager2.OnPageChangeCallback() {
            override fun onPageSelected(position: Int) {
                updatePageIndicators(position)
                updateButtons(position)
                refreshCurrentPagePermissionStatus()
            }
        })
    }

    private fun setupButtons() {
        binding.nextButton.setOnClickListener {
            val currentPage = pagerAdapter.getPage(binding.viewPager.currentItem)

            // Check if required permission is missing
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
            // Skip is only shown for optional permissions
            if (binding.viewPager.currentItem < pagerAdapter.itemCount - 1) {
                binding.viewPager.currentItem++
            }
        }

        updateButtons(0)
    }

    private fun setupPageIndicators() {
        val pageCount = pagerAdapter.itemCount
        val params = LinearLayout.LayoutParams(
            resources.getDimensionPixelSize(R.dimen.indicator_size),
            resources.getDimensionPixelSize(R.dimen.indicator_size)
        ).apply {
            marginStart = resources.getDimensionPixelSize(R.dimen.indicator_margin)
            marginEnd = resources.getDimensionPixelSize(R.dimen.indicator_margin)
        }

        for (i in 0 until pageCount) {
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
                if (index == position) R.drawable.indicator_active
                else R.drawable.indicator_inactive
            )
        }
    }

    private fun updateButtons(position: Int) {
        val page = pagerAdapter.getPage(position)
        val isLastPage = position == pagerAdapter.itemCount - 1

        binding.nextButton.text = if (isLastPage) {
            getString(R.string.get_started)
        } else {
            getString(R.string.next)
        }

        // Show skip button only for optional permission pages
        binding.skipButton.visibility = if (page.isOptional) {
            View.VISIBLE
        } else {
            View.GONE
        }
    }

    private fun requestPermissionForPage(page: OnboardingPage) {
        pendingPermissionPage = page

        when (page) {
            OnboardingPage.MICROPHONE -> {
                micPermissionLauncher.launch(Manifest.permission.RECORD_AUDIO)
            }
            OnboardingPage.NOTIFICATIONS -> {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
                } else {
                    // Notifications don't need runtime permission on older Android
                    handlePermissionResult(page, true)
                }
            }
            OnboardingPage.BLUETOOTH -> {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                    bluetoothPermissionLauncher.launch(Manifest.permission.BLUETOOTH_CONNECT)
                } else {
                    // Bluetooth Connect doesn't need runtime permission on older Android
                    handlePermissionResult(page, true)
                }
            }
            OnboardingPage.LOCATION -> {
                locationPermissionLauncher.launch(Manifest.permission.ACCESS_COARSE_LOCATION)
            }
            else -> { /* Not a permission page */ }
        }
    }

    private fun handlePermissionResult(page: OnboardingPage, granted: Boolean) {
        refreshCurrentPagePermissionStatus()

        if (granted) {
            Toast.makeText(this, R.string.permission_granted, Toast.LENGTH_SHORT).show()
        } else if (!page.isOptional) {
            Toast.makeText(this, R.string.permission_denied, Toast.LENGTH_SHORT).show()
        }
    }

    private fun refreshCurrentPagePermissionStatus() {
        val position = binding.viewPager.currentItem
        pagerAdapter.notifyItemChanged(position)
    }

    private fun isPermissionGranted(page: OnboardingPage): Boolean {
        return when (page) {
            OnboardingPage.MICROPHONE -> {
                checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED
            }
            OnboardingPage.NOTIFICATIONS -> {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED
                } else {
                    true // Always granted on older Android
                }
            }
            OnboardingPage.BLUETOOTH -> {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                    checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) == PackageManager.PERMISSION_GRANTED
                } else {
                    true // Always granted on older Android
                }
            }
            OnboardingPage.LOCATION -> {
                checkSelfPermission(Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED
            }
            else -> true // Not a permission page
        }
    }

    private fun completeOnboarding() {
        lifecycleScope.launch {
            preferencesManager.setOnboardingCompleted(true)
            startActivity(Intent(this@OnboardingActivity, MainActivity::class.java))
            finish()
        }
    }
}
