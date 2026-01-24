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

        loadMessages()
    }

    private fun loadMessages() {
        lifecycleScope.launch {
            try {
                val loadedMessages = repository.getMessages(conversationId)
                messages.clear()
                messages.addAll(loadedMessages)
                messageAdapter.notifyDataSetChanged()
                scrollToBottom()
            } catch (e: Exception) {
                Toast.makeText(this@ChatActivity, R.string.load_failed, Toast.LENGTH_SHORT).show()
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
        messages.add(tempUserMsg)
        messageAdapter.notifyItemInserted(messages.size - 1)
        scrollToBottom()

        val previousId = if (messages.size >= 2) messages[messages.size - 2].id else null

        lifecycleScope.launch {
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
                val tempIdx = messages.indexOfFirst { it.id == tempUserMsg.id }
                if (tempIdx >= 0) {
                    messages.removeAt(tempIdx)
                    messageAdapter.notifyItemRemoved(tempIdx)
                }
                Toast.makeText(this@ChatActivity, R.string.send_failed, Toast.LENGTH_SHORT).show()
            } finally {
                binding.sendButton.isEnabled = true
                binding.typingIndicator.visibility = View.GONE
            }
        }
    }

    private fun scrollToBottom() {
        if (messages.isNotEmpty()) {
            binding.messagesRecyclerView.scrollToPosition(messages.size - 1)
        }
    }

    private inner class MessageAdapter : RecyclerView.Adapter<RecyclerView.ViewHolder>() {

        companion object {
            private const val TYPE_USER = 0
            private const val TYPE_ASSISTANT = 1
        }

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
