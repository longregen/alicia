package com.alicia.assistant

import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.alicia.assistant.databinding.ActivityConversationsBinding
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.storage.ConversationRepository
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone

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
                binding.loadingState.visibility = View.GONE
                binding.emptyState.visibility = View.VISIBLE
                Toast.makeText(this@ConversationListActivity, R.string.load_failed, Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun createNewConversation() {
        lifecycleScope.launch {
            try {
                val conversation = repository.createConversation(getString(R.string.new_conversation))
                openConversation(conversation.id, conversation.title)
            } catch (e: Exception) {
                Toast.makeText(this@ConversationListActivity, R.string.create_failed, Toast.LENGTH_SHORT).show()
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

        private val dateParser = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }
        private val dateFormatter = SimpleDateFormat("MMM d, h:mm a", Locale.getDefault())

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
            val view = LayoutInflater.from(parent.context)
                .inflate(R.layout.item_conversation, parent, false)
            return ViewHolder(view)
        }

        override fun onBindViewHolder(holder: ViewHolder, position: Int) {
            holder.bind(getItem(position))
        }

        inner class ViewHolder(view: View) : RecyclerView.ViewHolder(view) {
            private val titleText: TextView = view.findViewById(R.id.conversationTitle)
            private val dateText: TextView = view.findViewById(R.id.conversationDate)

            fun bind(conversation: AliciaApiClient.Conversation) {
                titleText.text = conversation.title.ifBlank { "Untitled" }
                dateText.text = formatDate(conversation.updatedAt)
                itemView.setOnClickListener { onClick(conversation) }
            }

            private fun formatDate(isoDate: String): String {
                if (isoDate.isBlank()) return ""
                return try {
                    val normalized = isoDate.replace(Regex("\\.\\d+"), "").removeSuffix("Z") + "Z"
                    val date = dateParser.parse(normalized) ?: return isoDate
                    dateFormatter.format(date)
                } catch (e: Exception) {
                    isoDate.take(10)
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
