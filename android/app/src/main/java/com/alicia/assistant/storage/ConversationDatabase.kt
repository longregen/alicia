package com.alicia.assistant.storage

import android.content.ContentValues
import android.content.Context
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper
import com.alicia.assistant.service.AliciaApiClient

class ConversationDatabase(context: Context) :
    SQLiteOpenHelper(context, DB_NAME, null, DB_VERSION) {

    companion object {
        private const val DB_NAME = "alicia_conversations.db"
        private const val DB_VERSION = 1

        private const val TABLE_CONVERSATIONS = "conversations"
        private const val TABLE_MESSAGES = "messages"
        private const val TABLE_TOOL_USES = "tool_uses"
    }

    override fun onCreate(db: SQLiteDatabase) {
        db.execSQL("""
            CREATE TABLE $TABLE_CONVERSATIONS (
                id TEXT PRIMARY KEY,
                title TEXT NOT NULL DEFAULT '',
                status TEXT NOT NULL DEFAULT 'active',
                created_at TEXT NOT NULL DEFAULT '',
                updated_at TEXT NOT NULL DEFAULT ''
            )
        """.trimIndent())

        db.execSQL("""
            CREATE TABLE $TABLE_MESSAGES (
                id TEXT PRIMARY KEY,
                conversation_id TEXT NOT NULL,
                role TEXT NOT NULL,
                content TEXT NOT NULL DEFAULT '',
                status TEXT NOT NULL DEFAULT '',
                previous_id TEXT,
                FOREIGN KEY (conversation_id) REFERENCES $TABLE_CONVERSATIONS(id) ON DELETE CASCADE
            )
        """.trimIndent())

        db.execSQL("""
            CREATE TABLE $TABLE_TOOL_USES (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                message_id TEXT NOT NULL,
                tool_name TEXT NOT NULL,
                status TEXT NOT NULL DEFAULT '',
                FOREIGN KEY (message_id) REFERENCES $TABLE_MESSAGES(id) ON DELETE CASCADE
            )
        """.trimIndent())

        db.execSQL("CREATE INDEX idx_messages_conv ON $TABLE_MESSAGES(conversation_id)")
        db.execSQL("CREATE INDEX idx_tool_uses_msg ON $TABLE_TOOL_USES(message_id)")
    }

    override fun onUpgrade(db: SQLiteDatabase, oldVersion: Int, newVersion: Int) {
        db.execSQL("DROP TABLE IF EXISTS $TABLE_TOOL_USES")
        db.execSQL("DROP TABLE IF EXISTS $TABLE_MESSAGES")
        db.execSQL("DROP TABLE IF EXISTS $TABLE_CONVERSATIONS")
        onCreate(db)
    }

    override fun onConfigure(db: SQLiteDatabase) {
        super.onConfigure(db)
        db.setForeignKeyConstraintsEnabled(true)
    }

    fun cacheConversations(conversations: List<AliciaApiClient.Conversation>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            db.execSQL("DELETE FROM $TABLE_CONVERSATIONS")
            for (conv in conversations) {
                db.insert(TABLE_CONVERSATIONS, null, ContentValues().apply {
                    put("id", conv.id)
                    put("title", conv.title)
                    put("status", conv.status)
                    put("created_at", conv.createdAt)
                    put("updated_at", conv.updatedAt)
                })
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun getCachedConversations(): List<AliciaApiClient.Conversation> {
        val db = readableDatabase
        val cursor = db.query(
            TABLE_CONVERSATIONS, null, null, null, null, null,
            "updated_at DESC"
        )
        val results = mutableListOf<AliciaApiClient.Conversation>()
        cursor.use {
            while (it.moveToNext()) {
                results.add(AliciaApiClient.Conversation(
                    id = it.getString(it.getColumnIndexOrThrow("id")),
                    title = it.getString(it.getColumnIndexOrThrow("title")),
                    status = it.getString(it.getColumnIndexOrThrow("status")),
                    createdAt = it.getString(it.getColumnIndexOrThrow("created_at")),
                    updatedAt = it.getString(it.getColumnIndexOrThrow("updated_at"))
                ))
            }
        }
        return results
    }

    fun cacheConversation(conv: AliciaApiClient.Conversation) {
        val db = writableDatabase
        db.insertWithOnConflict(TABLE_CONVERSATIONS, null, ContentValues().apply {
            put("id", conv.id)
            put("title", conv.title)
            put("status", conv.status)
            put("created_at", conv.createdAt)
            put("updated_at", conv.updatedAt)
        }, SQLiteDatabase.CONFLICT_REPLACE)
    }

    fun cacheMessages(conversationId: String, messages: List<AliciaApiClient.Message>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            db.delete(TABLE_MESSAGES, "conversation_id = ?", arrayOf(conversationId))
            for (msg in messages) {
                insertMessage(db, msg)
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun appendMessage(msg: AliciaApiClient.Message) {
        val db = writableDatabase
        insertMessage(db, msg)
    }

    private fun insertMessage(db: SQLiteDatabase, msg: AliciaApiClient.Message) {
        db.insertWithOnConflict(TABLE_MESSAGES, null, ContentValues().apply {
            put("id", msg.id)
            put("conversation_id", msg.conversationId)
            put("role", msg.role)
            put("content", msg.content)
            put("status", msg.status)
            put("previous_id", msg.previousId)
        }, SQLiteDatabase.CONFLICT_REPLACE)

        db.delete(TABLE_TOOL_USES, "message_id = ?", arrayOf(msg.id))
        for (tu in msg.toolUses) {
            db.insert(TABLE_TOOL_USES, null, ContentValues().apply {
                put("message_id", msg.id)
                put("tool_name", tu.toolName)
                put("status", tu.status)
            })
        }
    }

    fun getCachedMessages(conversationId: String): List<AliciaApiClient.Message> {
        val db = readableDatabase
        val cursor = db.query(
            TABLE_MESSAGES, null,
            "conversation_id = ?", arrayOf(conversationId),
            null, null, "rowid ASC"
        )
        val results = mutableListOf<AliciaApiClient.Message>()
        cursor.use {
            while (it.moveToNext()) {
                val msgId = it.getString(it.getColumnIndexOrThrow("id"))
                val toolUses = getToolUses(db, msgId)
                results.add(AliciaApiClient.Message(
                    id = msgId,
                    conversationId = it.getString(it.getColumnIndexOrThrow("conversation_id")),
                    role = it.getString(it.getColumnIndexOrThrow("role")),
                    content = it.getString(it.getColumnIndexOrThrow("content")),
                    status = it.getString(it.getColumnIndexOrThrow("status")),
                    previousId = it.getString(it.getColumnIndexOrThrow("previous_id")),
                    toolUses = toolUses
                ))
            }
        }
        return results
    }

    private fun getToolUses(db: SQLiteDatabase, messageId: String): List<AliciaApiClient.ToolUseInfo> {
        val cursor = db.query(
            TABLE_TOOL_USES, null,
            "message_id = ?", arrayOf(messageId),
            null, null, null
        )
        val results = mutableListOf<AliciaApiClient.ToolUseInfo>()
        cursor.use {
            while (it.moveToNext()) {
                results.add(AliciaApiClient.ToolUseInfo(
                    toolName = it.getString(it.getColumnIndexOrThrow("tool_name")),
                    status = it.getString(it.getColumnIndexOrThrow("status"))
                ))
            }
        }
        return results
    }
}
