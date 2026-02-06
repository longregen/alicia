package com.alicia.assistant

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ImageView
import android.widget.TextView
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.alicia.assistant.model.VoskModelInfo
import com.alicia.assistant.service.ModelDownloadService
import com.alicia.assistant.service.VoiceAssistantService
import com.alicia.assistant.storage.PreferencesManager
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.LinearProgressIndicator
import com.google.android.material.radiobutton.MaterialRadioButton
import kotlinx.coroutines.launch
import java.io.File

class ModelManagerActivity : ComponentActivity() {

    private lateinit var preferencesManager: PreferencesManager
    private lateinit var adapter: ModelAdapter
    private var selectedModelId: String = "small-en-us"

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_model_manager)

        preferencesManager = PreferencesManager(this)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        toolbar.setNavigationOnClickListener { finish() }

        val recyclerView = findViewById<RecyclerView>(R.id.modelsRecyclerView)
        recyclerView.layoutManager = LinearLayoutManager(this)

        lifecycleScope.launch {
            selectedModelId = preferencesManager.getSettings().voskModelId
            adapter = ModelAdapter()
            recyclerView.adapter = adapter

            // Manual diff to issue targeted notifyItemChanged calls instead of notifyDataSetChanged
            var prev = emptyMap<String, Int>()
            ModelDownloadService.downloadState.collect { state ->
                val changed = (prev.keys + state.keys).filter { prev[it] != state[it] }
                prev = state
                val models = VoskModelInfo.entries
                for (key in changed) {
                    val idx = models.indexOfFirst { it.id == key }
                    if (idx >= 0) adapter.notifyItemChanged(idx)
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        VoiceAssistantService.ensureRunning(this)
    }

    private fun isModelDownloaded(modelInfo: VoskModelInfo): Boolean {
        val modelDir = File(filesDir, "vosk-models/${modelInfo.id}")
        if (!modelDir.exists()) return false
        if (File(modelDir, ".extracting").exists()) return false
        return modelDir.listFiles()?.isNotEmpty() == true
    }

    private fun selectModel(modelInfo: VoskModelInfo) {
        if (!isModelDownloaded(modelInfo)) return
        selectedModelId = modelInfo.id
        adapter.notifyDataSetChanged()
        lifecycleScope.launch {
            val settings = preferencesManager.getSettings()
            preferencesManager.saveSettings(settings.copy(voskModelId = modelInfo.id))
        }
    }

    private fun downloadModel(modelInfo: VoskModelInfo) {
        if (ModelDownloadService.isDownloading(modelInfo.id)) return
        ModelDownloadService.start(this, modelInfo.id)
    }

    inner class ModelViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        val radioButton: MaterialRadioButton = view.findViewById(R.id.modelRadioButton)
        val modelName: TextView = view.findViewById(R.id.modelName)
        val modelSize: TextView = view.findViewById(R.id.modelSize)
        val downloadButton: MaterialButton = view.findViewById(R.id.downloadButton)
        val downloadProgress: LinearProgressIndicator = view.findViewById(R.id.downloadProgress)
        val downloadedIcon: ImageView = view.findViewById(R.id.downloadedIcon)
    }

    inner class ModelAdapter : RecyclerView.Adapter<ModelViewHolder>() {
        private val models = VoskModelInfo.entries.toList()

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ModelViewHolder {
            val view = LayoutInflater.from(parent.context)
                .inflate(R.layout.item_model, parent, false)
            return ModelViewHolder(view)
        }

        override fun onBindViewHolder(holder: ModelViewHolder, position: Int) {
            val model = models[position]
            val downloaded = isModelDownloaded(model)
            val isSelected = model.id == selectedModelId
            val progress = ModelDownloadService.downloadState.value[model.id]

            holder.modelName.text = model.displayName
            holder.modelSize.text = "${model.sizeMb} MB"
            holder.radioButton.isChecked = isSelected
            holder.radioButton.isEnabled = downloaded

            when {
                progress != null -> {
                    holder.downloadButton.visibility = View.GONE
                    holder.downloadProgress.visibility = View.VISIBLE
                    holder.downloadedIcon.visibility = View.GONE
                    if (progress == -1) {
                        holder.downloadProgress.isIndeterminate = true
                    } else {
                        holder.downloadProgress.isIndeterminate = false
                        holder.downloadProgress.progress = progress
                    }
                }
                downloaded -> {
                    holder.downloadButton.visibility = View.GONE
                    holder.downloadProgress.visibility = View.GONE
                    holder.downloadedIcon.visibility = View.VISIBLE
                }
                else -> {
                    holder.downloadButton.visibility = View.VISIBLE
                    holder.downloadProgress.visibility = View.GONE
                    holder.downloadedIcon.visibility = View.GONE
                }
            }

            holder.radioButton.setOnClickListener {
                if (downloaded) selectModel(model)
            }
            holder.itemView.setOnClickListener {
                if (downloaded) selectModel(model)
            }
            holder.downloadButton.setOnClickListener {
                downloadModel(model)
            }
        }

        override fun getItemCount() = models.size
    }
}
