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
    val isOptional: Boolean = false
) {
    WELCOME(R.layout.onboarding_page_welcome),
    MICROPHONE(R.layout.onboarding_page_permission, isPermissionPage = true),
    NOTIFICATIONS(R.layout.onboarding_page_permission, isPermissionPage = true),
    BLUETOOTH(R.layout.onboarding_page_permission, isPermissionPage = true, isOptional = true),
    LOCATION(R.layout.onboarding_page_permission, isPermissionPage = true, isOptional = true),
    COMPLETE(R.layout.onboarding_page_complete)
}

data class PermissionPageConfig(
    val iconRes: Int,
    val titleRes: Int,
    val descRes: Int
)

class OnboardingPagerAdapter(
    private val onGrantPermission: (OnboardingPage) -> Unit,
    private val getPermissionStatus: (OnboardingPage) -> Boolean
) : RecyclerView.Adapter<OnboardingPagerAdapter.PageViewHolder>() {

    private val pages = OnboardingPage.entries.toList()

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
        )
    )

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): PageViewHolder {
        val page = OnboardingPage.entries[viewType]
        val view = LayoutInflater.from(parent.context).inflate(page.layoutRes, parent, false)
        return PageViewHolder(view)
    }

    override fun onBindViewHolder(holder: PageViewHolder, position: Int) {
        val page = pages[position]
        if (page.isPermissionPage) {
            bindPermissionPage(holder, page)
        }
    }

    private fun bindPermissionPage(holder: PageViewHolder, page: OnboardingPage) {
        val config = permissionConfigs[page] ?: return

        holder.itemView.findViewById<ImageView>(R.id.iconImage)?.setImageResource(config.iconRes)
        holder.itemView.findViewById<TextView>(R.id.titleText)?.setText(config.titleRes)
        holder.itemView.findViewById<TextView>(R.id.descriptionText)?.setText(config.descRes)

        val grantButton = holder.itemView.findViewById<MaterialButton>(R.id.grantButton)
        val statusText = holder.itemView.findViewById<TextView>(R.id.statusText)

        updatePermissionUI(grantButton, statusText, page)

        grantButton?.setOnClickListener {
            onGrantPermission(page)
        }
    }

    private fun updatePermissionUI(
        grantButton: MaterialButton?,
        statusText: TextView?,
        page: OnboardingPage
    ) {
        val granted = getPermissionStatus(page)

        if (granted) {
            grantButton?.visibility = View.GONE
            statusText?.visibility = View.VISIBLE
            statusText?.setText(R.string.permission_granted)
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
