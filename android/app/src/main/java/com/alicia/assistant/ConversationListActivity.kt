package com.alicia.assistant

import android.content.Intent
import android.content.res.Configuration
import android.graphics.Typeface
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.alicia.assistant.databinding.ActivityConversationsBinding
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.storage.ConversationRepository
import com.alicia.assistant.telemetry.AliciaTelemetry
import com.google.android.material.card.MaterialCardView
import io.opentelemetry.api.common.Attributes
import kotlinx.coroutines.launch
import java.time.Duration
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

class ConversationListActivity : ComponentActivity() {

    private lateinit var binding: ActivityConversationsBinding
    private lateinit var repository: ConversationRepository
    private val adapter = ConversationAdapter { conversation ->
        openConversation(conversation.id, conversation.title)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityConversationsBinding.inflate(layoutInflater)
        setContentView(binding.root)

        val apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
        repository = ConversationRepository(this, apiClient)

        binding.toolbar.setNavigationOnClickListener { finish() }

        binding.conversationsRecyclerView.layoutManager = LinearLayoutManager(this)
        binding.conversationsRecyclerView.adapter = adapter

        binding.newConversationFab.setOnClickListener {
            createNewConversation()
        }
    }

    override fun onResume() {
        super.onResume()
        loadConversations()
    }

    private fun loadConversations() {
        lifecycleScope.launch {
            AliciaTelemetry.withSpanAsync("conversations.load") { span ->
                try {
                    val conversations = repository.listConversations()
                    binding.loadingState.visibility = View.GONE

                    if (conversations.isEmpty()) {
                        binding.emptyState.visibility = View.VISIBLE
                        binding.conversationsRecyclerView.visibility = View.GONE
                    } else {
                        binding.emptyState.visibility = View.GONE
                        binding.conversationsRecyclerView.visibility = View.VISIBLE
                        adapter.submitList(conversations)
                    }
                } catch (e: Exception) {
                    AliciaTelemetry.recordError(span, e)
                    binding.loadingState.visibility = View.GONE
                    binding.emptyState.visibility = View.VISIBLE
                    Toast.makeText(this@ConversationListActivity, R.string.load_failed, Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun createNewConversation() {
        lifecycleScope.launch {
            AliciaTelemetry.withSpanAsync("conversations.create") { span ->
                try {
                    val conversation = repository.createConversation(getString(R.string.new_conversation))
                    openConversation(conversation.id, conversation.title)
                } catch (e: Exception) {
                    AliciaTelemetry.recordError(span, e)
                    Toast.makeText(this@ConversationListActivity, R.string.create_failed, Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun openConversation(id: String, title: String) {
        val intent = Intent(this, ChatActivity::class.java).apply {
            putExtra(ChatActivity.EXTRA_CONVERSATION_ID, id)
            putExtra(ChatActivity.EXTRA_CONVERSATION_TITLE, title)
        }
        startActivity(intent)
    }

    private class ConversationAdapter(
        private val onClick: (AliciaApiClient.Conversation) -> Unit
    ) : ListAdapter<AliciaApiClient.Conversation, ConversationAdapter.ViewHolder>(DiffCallback) {

        private val dateFormatter = DateTimeFormatter.ofPattern("MMM d, h:mm a", Locale.getDefault())
            .withZone(ZoneId.systemDefault())

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
            val view = LayoutInflater.from(parent.context)
                .inflate(R.layout.item_conversation, parent, false)
            return ViewHolder(view)
        }

        override fun onBindViewHolder(holder: ViewHolder, position: Int) {
            holder.bind(getItem(position), position)
        }

        inner class ViewHolder(view: View) : RecyclerView.ViewHolder(view) {
            private val card: MaterialCardView = view.findViewById(R.id.conversationCard)
            private val titleText: TextView = view.findViewById(R.id.conversationTitle)
            private val dateText: TextView = view.findViewById(R.id.conversationDate)
            private val recentIndicator: View = view.findViewById(R.id.recentIndicator)

            fun bind(conversation: AliciaApiClient.Conversation, position: Int) {
                val context = itemView.context
                val isDarkMode = (context.resources.configuration.uiMode and
                    Configuration.UI_MODE_NIGHT_MASK) == Configuration.UI_MODE_NIGHT_YES

                titleText.text = conversation.title.ifBlank { "Untitled" }
                dateText.text = formatDate(conversation.updatedAt)
                itemView.setOnClickListener { onClick(conversation) }

                // Determine recency
                val isRecent = isWithinLast24Hours(conversation.updatedAt)
                val isMostRecent = position == 0

                // Apply visual distinctions
                when {
                    isMostRecent -> {
                        // Most recent: highlighted background + bold title + indicator
                        val bgColor = if (isDarkMode) {
                            ContextCompat.getColor(context, R.color.conversation_most_recent_background_dark)
                        } else {
                            ContextCompat.getColor(context, R.color.conversation_most_recent_background_light)
                        }
                        card.setCardBackgroundColor(bgColor)
                        titleText.setTypeface(null, Typeface.BOLD)
                        recentIndicator.visibility = View.VISIBLE
                    }
                    isRecent -> {
                        // Recent (within 24h): subtle background + semi-bold + indicator
                        val bgColor = if (isDarkMode) {
                            ContextCompat.getColor(context, R.color.conversation_recent_background_dark)
                        } else {
                            ContextCompat.getColor(context, R.color.conversation_recent_background_light)
                        }
                        card.setCardBackgroundColor(bgColor)
                        titleText.setTypeface(null, Typeface.BOLD)
                        recentIndicator.visibility = View.VISIBLE
                    }
                    else -> {
                        // Older conversations: default style
                        card.setCardBackgroundColor(
                            ContextCompat.getColor(context, android.R.color.transparent)
                        )
                        titleText.setTypeface(null, Typeface.NORMAL)
                        recentIndicator.visibility = View.GONE
                    }
                }
            }

            private fun formatDate(isoDate: String): String {
                if (isoDate.isBlank()) return ""
                return try {
                    val instant = Instant.parse(isoDate)
                    dateFormatter.format(instant)
                } catch (e: Exception) {
                    isoDate.take(10)
                }
            }

            private fun isWithinLast24Hours(isoDate: String): Boolean {
                if (isoDate.isBlank()) return false
                return try {
                    val instant = Instant.parse(isoDate)
                    val now = Instant.now()
                    Duration.between(instant, now).toHours() < 24
                } catch (e: Exception) {
                    false
                }
            }
        }

        companion object DiffCallback : DiffUtil.ItemCallback<AliciaApiClient.Conversation>() {
            override fun areItemsTheSame(
                oldItem: AliciaApiClient.Conversation,
                newItem: AliciaApiClient.Conversation
            ): Boolean = oldItem.id == newItem.id

            override fun areContentsTheSame(
                oldItem: AliciaApiClient.Conversation,
                newItem: AliciaApiClient.Conversation
            ): Boolean = oldItem == newItem
        }
    }
}
