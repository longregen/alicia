package com.alicia.assistant

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ImageView
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.google.android.material.button.MaterialButton

enum class OnboardingPage(
    val layoutRes: Int,
    val isPermissionPage: Boolean = false,
    val isOptional: Boolean = false,
    val isAssistantPage: Boolean = false
) {
    WELCOME(R.layout.onboarding_page_welcome),
    MICROPHONE(R.layout.onboarding_page_permission, isPermissionPage = true),
    NOTIFICATIONS(R.layout.onboarding_page_permission, isPermissionPage = true),
    BLUETOOTH(R.layout.onboarding_page_permission, isPermissionPage = true, isOptional = true),
    LOCATION(R.layout.onboarding_page_permission, isPermissionPage = true, isOptional = true),
    ASSISTANT(R.layout.onboarding_page_permission, isAssistantPage = true, isOptional = true),
    COMPLETE(R.layout.onboarding_page_complete)
}

data class PermissionPageConfig(
    val iconRes: Int,
    val titleRes: Int,
    val descRes: Int
)

class OnboardingPagerAdapter(
    private val onGrantPermission: (OnboardingPage) -> Unit,
    private val getPermissionStatus: (OnboardingPage) -> Boolean,
    private val onSetupAssistant: () -> Unit = {},
    private val isAssistantConfigured: () -> Boolean = { false }
) : RecyclerView.Adapter<OnboardingPagerAdapter.PageViewHolder>() {

    private var pages = computePages()

    private fun computePages() = OnboardingPage.entries.filter { page ->
        when {
            page.isPermissionPage -> !getPermissionStatus(page)
            page.isAssistantPage -> !isAssistantConfigured()
            else -> true
        }
    }

    fun refreshPages() {
        pages = computePages()
        notifyDataSetChanged()
    }

    private val permissionConfigs = mapOf(
        OnboardingPage.MICROPHONE to PermissionPageConfig(
            R.drawable.ic_microphone,
            R.string.onboarding_mic_title,
            R.string.onboarding_mic_desc
        ),
        OnboardingPage.NOTIFICATIONS to PermissionPageConfig(
            R.drawable.ic_notifications,
            R.string.onboarding_notifications_title,
            R.string.onboarding_notifications_desc
        ),
        OnboardingPage.BLUETOOTH to PermissionPageConfig(
            R.drawable.ic_bluetooth,
            R.string.onboarding_bluetooth_title,
            R.string.onboarding_bluetooth_desc
        ),
        OnboardingPage.LOCATION to PermissionPageConfig(
            R.drawable.ic_location,
            R.string.onboarding_location_title,
            R.string.onboarding_location_desc
        ),
        OnboardingPage.ASSISTANT to PermissionPageConfig(
            R.drawable.ic_assistant,
            R.string.onboarding_assistant_title,
            R.string.onboarding_assistant_desc
        )
    )

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): PageViewHolder {
        val page = pages[viewType]
        val view = LayoutInflater.from(parent.context).inflate(page.layoutRes, parent, false)
        return PageViewHolder(view)
    }

    override fun onBindViewHolder(holder: PageViewHolder, position: Int) {
        val page = pages[position]
        if (page.isPermissionPage) {
            bindPermissionPage(holder, page)
        } else if (page.isAssistantPage) {
            bindAssistantPage(holder, page)
        }
    }

    private fun bindPermissionPage(holder: PageViewHolder, page: OnboardingPage) {
        val config = permissionConfigs[page] ?: return

        holder.itemView.findViewById<ImageView>(R.id.iconImage)?.setImageResource(config.iconRes)
        holder.itemView.findViewById<TextView>(R.id.titleText)?.setText(config.titleRes)
        holder.itemView.findViewById<TextView>(R.id.descriptionText)?.setText(config.descRes)

        val grantButton = holder.itemView.findViewById<MaterialButton>(R.id.grantButton)
        val statusText = holder.itemView.findViewById<TextView>(R.id.statusText)

        val granted = getPermissionStatus(page)
        updateStatusUI(grantButton, statusText, granted, R.string.permission_granted)

        grantButton?.setOnClickListener {
            onGrantPermission(page)
        }
    }

    private fun bindAssistantPage(holder: PageViewHolder, page: OnboardingPage) {
        val config = permissionConfigs[page] ?: return

        holder.itemView.findViewById<ImageView>(R.id.iconImage)?.setImageResource(config.iconRes)
        holder.itemView.findViewById<TextView>(R.id.titleText)?.setText(config.titleRes)
        holder.itemView.findViewById<TextView>(R.id.descriptionText)?.setText(config.descRes)

        val grantButton = holder.itemView.findViewById<MaterialButton>(R.id.grantButton)
        val statusText = holder.itemView.findViewById<TextView>(R.id.statusText)

        grantButton?.setText(R.string.open_settings)

        val configured = isAssistantConfigured()
        updateStatusUI(grantButton, statusText, configured, R.string.assistant_configured)

        grantButton?.setOnClickListener {
            onSetupAssistant()
        }
    }

    private fun updateStatusUI(
        grantButton: MaterialButton?,
        statusText: TextView?,
        isComplete: Boolean,
        completeTextRes: Int
    ) {
        if (isComplete) {
            grantButton?.visibility = View.GONE
            statusText?.visibility = View.VISIBLE
            statusText?.setText(completeTextRes)
            statusText?.setTextColor(statusText.context.getColor(android.R.color.holo_green_dark))
        } else {
            grantButton?.visibility = View.VISIBLE
            statusText?.visibility = View.GONE
        }
    }

    override fun getItemCount(): Int = pages.size

    override fun getItemViewType(position: Int): Int = position

    fun getPage(position: Int): OnboardingPage = pages[position]

    class PageViewHolder(itemView: View) : RecyclerView.ViewHolder(itemView)
}
