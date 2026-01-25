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
import com.alicia.assistant.ws.AssistantWebSocket
import com.alicia.assistant.ws.MessageListener
import io.opentelemetry.api.common.Attributes
import com.google.android.material.chip.Chip
import com.google.android.material.chip.ChipGroup
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

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
    private var webSocket: AssistantWebSocket? = null
    private var pendingMessageId: String? = null

    private val messageListener = object : MessageListener {
        override fun onUserMessage(conversationId: String, messageId: String, content: String, previousId: String?) {
            // Server confirmed user message - update the temp message with the real ID
            if (conversationId == this@ChatActivity.conversationId) {
                runOnUiThread {
                    pendingMessageId?.let { pendingId ->
                        val idx = messages.indexOfFirst { it.id == pendingId }
                        if (idx >= 0) {
                            messages[idx] = AliciaApiClient.Message(
                                id = messageId,
                                conversationId = conversationId,
                                role = "user",
                                content = content,
                                status = "completed",
                                previousId = previousId
                            )
                            messageAdapter.notifyItemChanged(idx)
                        }
                    }
                }
            }
        }

        override fun onAssistantMessage(conversationId: String, messageId: String, content: String, previousId: String?) {
            if (conversationId == this@ChatActivity.conversationId) {
                runOnUiThread {
                    val msg = AliciaApiClient.Message(
                        id = messageId,
                        conversationId = conversationId,
                        role = "assistant",
                        content = content,
                        status = "completed",
                        previousId = previousId
                    )
                    messages.add(msg)
                    messageAdapter.notifyItemInserted(messages.size - 1)
                    scrollToBottom()
                    binding.sendButton.isEnabled = true
                    binding.typingIndicator.visibility = View.GONE
                    pendingMessageId = null
                }
            }
        }

        override fun onThinkingUpdate(conversationId: String, messageId: String, content: String, progress: Float) {
            // Could update typing indicator with progress
        }

        override fun onToolUseStarted(conversationId: String, toolName: String, arguments: Map<String, Any?>) {
            // Could show tool usage indicator
        }

        override fun onToolUseCompleted(conversationId: String, toolName: String, success: Boolean) {
            // Could update tool usage indicator
        }

        override fun onError(conversationId: String, error: String) {
            if (conversationId == this@ChatActivity.conversationId) {
                runOnUiThread {
                    Toast.makeText(this@ChatActivity, error, Toast.LENGTH_SHORT).show()
                    binding.sendButton.isEnabled = true
                    binding.typingIndicator.visibility = View.GONE
                    // Remove pending user message on error
                    pendingMessageId?.let { pendingId ->
                        val idx = messages.indexOfFirst { it.id == pendingId }
                        if (idx >= 0) {
                            messages.removeAt(idx)
                            messageAdapter.notifyItemRemoved(idx)
                        }
                    }
                    pendingMessageId = null
                }
            }
        }
    }

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

        webSocket = (application as? AliciaApplication)?.assistantWebSocket
        webSocket?.let { ws ->
            ws.addMessageListener(conversationId, messageListener)
            ws.subscribeToConversation(conversationId)
        }

        binding.toolbar.setNavigationOnClickListener { finish() }

        val layoutManager = LinearLayoutManager(this).apply {
            stackFromEnd = true
        }
        binding.messagesRecyclerView.layoutManager = layoutManager
        binding.messagesRecyclerView.adapter = messageAdapter

        binding.sendButton.setOnClickListener { sendMessage() }

        loadMessages()
    }

    override fun onDestroy() {
        webSocket?.removeMessageListener(conversationId)
        webSocket?.unsubscribeFromConversation(conversationId)
        super.onDestroy()
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
                    scrollToBottom()
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
            status = "complete"
        )
        pendingMessageId = tempUserMsg.id
        messages.add(tempUserMsg)
        messageAdapter.notifyItemInserted(messages.size - 1)
        scrollToBottom()

        val previousId = if (messages.size >= 2) messages[messages.size - 2].id else null

        // Try WebSocket first
        val ws = webSocket
        if (ws != null && ws.isConnected()) {
            lifecycleScope.launch {
                AliciaTelemetry.withSpanAsync("chat.send_message_ws", Attributes.builder()
                    .put("conversation.id", conversationId)
                    .put("message.content_length", content.length.toLong())
                    .build()
                ) { span ->
                    val sent = withContext(Dispatchers.IO) {
                        ws.sendUserMessage(conversationId, content, previousId)
                    }
                    if (!sent) {
                        AliciaTelemetry.addSpanEvent(span, "ws_send_failed_fallback_http")
                        // Fall back to HTTP if WebSocket send fails
                        sendMessageViaHttp(content, tempUserMsg, previousId)
                    }
                    // Response will come via messageListener callback
                }
            }
        } else {
            // Fall back to HTTP
            sendMessageViaHttp(content, tempUserMsg, previousId)
        }
    }

    private fun sendMessageViaHttp(content: String, tempUserMsg: AliciaApiClient.Message, previousId: String?) {
        lifecycleScope.launch {
            AliciaTelemetry.withSpanAsync("chat.send_message_http", Attributes.builder()
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
                } catch (e: Exception) {
                    AliciaTelemetry.recordError(span, e)
                    val tempIdx = messages.indexOfFirst { it.id == tempUserMsg.id }
                    if (tempIdx >= 0) {
                        messages.removeAt(tempIdx)
                        messageAdapter.notifyItemRemoved(tempIdx)
                    }
                    Toast.makeText(this@ChatActivity, R.string.send_failed, Toast.LENGTH_SHORT).show()
                } finally {
                    binding.sendButton.isEnabled = true
                    binding.typingIndicator.visibility = View.GONE
                    pendingMessageId = null
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
