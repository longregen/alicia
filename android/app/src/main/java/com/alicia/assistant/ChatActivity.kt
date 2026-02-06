package com.alicia.assistant

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.alicia.assistant.databinding.ActivityChatBinding
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.storage.ConversationRepository
import com.alicia.assistant.telemetry.AliciaTelemetry
import io.opentelemetry.api.common.Attributes
import com.google.android.material.chip.Chip
import com.google.android.material.chip.ChipGroup
import kotlinx.coroutines.launch

class ChatActivity : ComponentActivity() {

    companion object {
        const val EXTRA_CONVERSATION_ID = "conversation_id"
        const val EXTRA_CONVERSATION_TITLE = "conversation_title"
    }

    private lateinit var binding: ActivityChatBinding
    private lateinit var repository: ConversationRepository
    private lateinit var conversationId: String
    private val messageAdapter = MessageAdapter()
    private val messages = mutableListOf<AliciaApiClient.Message>()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityChatBinding.inflate(layoutInflater)
        setContentView(binding.root)

        conversationId = intent.getStringExtra(EXTRA_CONVERSATION_ID) ?: run {
            finish()
            return
        }
        val title = intent.getStringExtra(EXTRA_CONVERSATION_TITLE) ?: getString(R.string.chat)
        binding.toolbar.title = title

        val apiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
        repository = ConversationRepository(this, apiClient)

        binding.toolbar.setNavigationOnClickListener { finish() }

        val layoutManager = LinearLayoutManager(this).apply {
            stackFromEnd = true
        }
        binding.messagesRecyclerView.layoutManager = layoutManager
        binding.messagesRecyclerView.adapter = messageAdapter

        binding.sendButton.setOnClickListener { sendMessage() }

        setupConversationStarters()
        loadMessages()
    }

    private fun setupConversationStarters() {
        val starterClickListener = View.OnClickListener { view ->
            val text = when (view.id) {
                R.id.starter1 -> getString(R.string.starter_explain)
                R.id.starter2 -> getString(R.string.starter_help)
                R.id.starter3 -> getString(R.string.starter_summarize)
                R.id.starter4 -> getString(R.string.starter_brainstorm)
                else -> return@OnClickListener
            }
            binding.messageInput.setText(text)
            binding.messageInput.setSelection(text.length)
            binding.messageInput.requestFocus()
        }

        binding.starter1.setOnClickListener(starterClickListener)
        binding.starter2.setOnClickListener(starterClickListener)
        binding.starter3.setOnClickListener(starterClickListener)
        binding.starter4.setOnClickListener(starterClickListener)
    }

    private fun updateEmptyState() {
        if (messages.isEmpty()) {
            binding.emptyStateContainer.visibility = View.VISIBLE
            binding.messagesRecyclerView.visibility = View.GONE
        } else {
            binding.emptyStateContainer.visibility = View.GONE
            binding.messagesRecyclerView.visibility = View.VISIBLE
        }
    }

    private fun loadMessages() {
        lifecycleScope.launch {
            AliciaTelemetry.withSpanAsync("chat.load_messages", Attributes.builder()
                .put("conversation.id", conversationId)
                .build()
            ) { span ->
                try {
                    val loadedMessages = repository.getMessages(conversationId)
                    messages.clear()
                    messages.addAll(loadedMessages)
                    messageAdapter.notifyDataSetChanged()
                    updateEmptyState()
                    scrollToBottom()
                } catch (e: AliciaApiClient.ApiException) {
                    AliciaTelemetry.recordError(span, e)
                    if (e.statusCode == 404) {
                        Toast.makeText(this@ChatActivity, R.string.conversation_not_found, Toast.LENGTH_SHORT).show()
                        finish()
                    } else {
                        Toast.makeText(this@ChatActivity, R.string.load_failed, Toast.LENGTH_SHORT).show()
                    }
                } catch (e: Exception) {
                    AliciaTelemetry.recordError(span, e)
                    Toast.makeText(this@ChatActivity, R.string.load_failed, Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun sendMessage() {
        val content = binding.messageInput.text?.toString()?.trim() ?: return
        if (content.isEmpty()) return

        binding.messageInput.text?.clear()
        binding.sendButton.isEnabled = false
        binding.typingIndicator.visibility = View.VISIBLE

        val tempUserMsg = AliciaApiClient.Message(
            id = "temp_${System.currentTimeMillis()}",
            conversationId = conversationId,
            role = "user",
            content = content,
            status = "pending"
        )
        val previousId = if (messages.isNotEmpty()) messages[messages.size - 1].id else null

        messages.add(tempUserMsg)
        messageAdapter.notifyItemInserted(messages.size - 1)
        updateEmptyState()
        scrollToBottom()

        lifecycleScope.launch {
            AliciaTelemetry.withSpanAsync("chat.send_message", Attributes.builder()
                .put("conversation.id", conversationId)
                .put("message.content_length", content.length.toLong())
                .build()
            ) { span ->
                try {
                    val response = repository.sendMessage(conversationId, content, previousId)

                    val tempIdx = messages.indexOfFirst { it.id == tempUserMsg.id }
                    if (tempIdx >= 0) {
                        messages[tempIdx] = response.userMessage
                        messageAdapter.notifyItemChanged(tempIdx)
                    }

                    messages.add(response.assistantMessage)
                    messageAdapter.notifyItemInserted(messages.size - 1)
                    scrollToBottom()

                    // Update toolbar title if server provided a new one
                    response.conversationTitle?.let { newTitle ->
                        binding.toolbar.title = newTitle
                    }
                } catch (e: Exception) {
                    AliciaTelemetry.recordError(span, e)
                    val tempIdx = messages.indexOfFirst { it.id == tempUserMsg.id }
                    if (tempIdx >= 0) {
                        messages.removeAt(tempIdx)
                        messageAdapter.notifyItemRemoved(tempIdx)
                        updateEmptyState()
                    }
                    Toast.makeText(this@ChatActivity, R.string.send_failed, Toast.LENGTH_SHORT).show()
                } finally {
                    binding.sendButton.isEnabled = true
                    binding.typingIndicator.visibility = View.GONE
                }
            }
        }
    }

    private fun scrollToBottom() {
        if (messages.isNotEmpty()) {
            binding.messagesRecyclerView.scrollToPosition(messages.size - 1)
        }
    }

    private inner class MessageAdapter : RecyclerView.Adapter<RecyclerView.ViewHolder>() {

        private val TYPE_USER = 0
        private val TYPE_ASSISTANT = 1

        override fun getItemViewType(position: Int): Int {
            return if (messages[position].role == "user") TYPE_USER else TYPE_ASSISTANT
        }

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): RecyclerView.ViewHolder {
            return if (viewType == TYPE_USER) {
                val view = LayoutInflater.from(parent.context)
                    .inflate(R.layout.item_message_user, parent, false)
                UserMessageViewHolder(view)
            } else {
                val view = LayoutInflater.from(parent.context)
                    .inflate(R.layout.item_message_assistant, parent, false)
                AssistantMessageViewHolder(view)
            }
        }

        override fun onBindViewHolder(holder: RecyclerView.ViewHolder, position: Int) {
            val message = messages[position]
            when (holder) {
                is UserMessageViewHolder -> holder.bind(message)
                is AssistantMessageViewHolder -> holder.bind(message)
            }
        }

        override fun getItemCount() = messages.size
    }

    private class UserMessageViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        private val contentText: TextView = view.findViewById(R.id.messageContent)

        fun bind(message: AliciaApiClient.Message) {
            contentText.text = message.content
        }
    }

    private class AssistantMessageViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        private val contentText: TextView = view.findViewById(R.id.messageContent)
        private val chipGroup: ChipGroup = view.findViewById(R.id.toolChipGroup)

        fun bind(message: AliciaApiClient.Message) {
            contentText.text = message.content

            if (message.toolUses.isNotEmpty()) {
                chipGroup.visibility = View.VISIBLE
                chipGroup.removeAllViews()
                for (toolUse in message.toolUses) {
                    val chip = Chip(chipGroup.context).apply {
                        text = chipGroup.context.getString(R.string.tool_used, toolUse.toolName)
                        isClickable = false
                        isCheckable = false
                        textSize = 11f
                        chipMinHeight = 24f
                    }
                    chipGroup.addView(chip)
                }
            } else {
                chipGroup.visibility = View.GONE
            }
        }
    }
}
