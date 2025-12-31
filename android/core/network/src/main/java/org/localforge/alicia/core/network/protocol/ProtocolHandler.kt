package org.localforge.alicia.core.network.protocol

import org.localforge.alicia.core.network.protocol.bodies.*
import timber.log.Timber
import org.msgpack.core.MessagePack
import org.msgpack.core.MessageUnpacker
import java.io.ByteArrayOutputStream

/**
 * Handles MessagePack encoding and decoding of protocol envelopes
 */
class ProtocolHandler {

    /**
     * Encode an envelope to MessagePack bytes
     */
    fun encode(envelope: Envelope): ByteArray {
        val output = ByteArrayOutputStream()
        val packer = MessagePack.newDefaultPacker(output)

        try {
            // Pack envelope as map with exactly 5 fields per protocol spec: stanzaId, conversationId, type, meta, body
            packer.packMapHeader(5)

            // stanzaId
            packer.packString("stanzaId")
            packer.packInt(envelope.stanzaId)

            // conversationId
            packer.packString("conversationId")
            packer.packString(envelope.conversationId)

            // type
            packer.packString("type")
            packer.packInt(envelope.type.value)

            // meta (optional)
            packer.packString("meta")
            if (envelope.meta != null) {
                packMap(packer, envelope.meta)
            } else {
                packer.packNil()
            }

            // body
            packer.packString("body")
            packBody(packer, envelope.type, envelope.body)

            packer.flush()
            return output.toByteArray()
        } catch (e: Exception) {
            Timber.e(e, "Failed to encode envelope")
            throw e
        } finally {
            packer.close()
        }
    }

    /**
     * Decode MessagePack bytes to an envelope
     */
    fun decode(data: ByteArray): Envelope {
        val unpacker = MessagePack.newDefaultUnpacker(data)

        try {
            // Unpack envelope map
            val mapSize = unpacker.unpackMapHeader()
            if (mapSize != 5) {
                throw IllegalArgumentException("Invalid envelope: expected 5 fields, got $mapSize")
            }

            var stanzaId: Int? = null
            var conversationId: String? = null
            var type: MessageType? = null
            var meta: Map<String, Any?>? = null
            var body: Any? = null

            for (i in 0 until mapSize) {
                when (unpacker.unpackString()) {
                    "stanzaId" -> stanzaId = unpacker.unpackInt()
                    "conversationId" -> conversationId = unpacker.unpackString()
                    "type" -> type = MessageType.fromInt(unpacker.unpackInt())
                    "meta" -> meta = if (unpacker.tryUnpackNil()) null else unpackMap(unpacker)
                    "body" -> {
                        val messageType = type ?: throw IllegalArgumentException("Type must come before body")
                        body = unpackBody(unpacker, messageType)
                    }
                }
            }
        // Filter out null values from meta to match Envelope's Map<String, Any> type
        val filteredMeta = meta?.filterValues { it != null }?.mapValues { it.value!! }


            return Envelope(
                stanzaId = stanzaId ?: throw IllegalArgumentException("Missing stanzaId"),
                conversationId = conversationId ?: throw IllegalArgumentException("Missing conversationId"),
                type = type ?: throw IllegalArgumentException("Missing type"),
                meta = filteredMeta,
                body = body ?: throw IllegalArgumentException("Missing body")
            )
        } catch (e: Exception) {
            Timber.e(e, "Failed to decode envelope")
            throw e
        } finally {
            unpacker.close()
        }
    }

    private fun packBody(packer: org.msgpack.core.MessagePacker, type: MessageType, body: Any) {
        when (type) {
            MessageType.ERROR_MESSAGE -> packErrorMessage(packer, body as ErrorMessageBody)
            MessageType.USER_MESSAGE -> packUserMessage(packer, body as UserMessageBody)
            MessageType.ASSISTANT_MESSAGE -> packAssistantMessage(packer, body as AssistantMessageBody)
            MessageType.TRANSCRIPTION -> packTranscription(packer, body as TranscriptionBody)
            MessageType.START_ANSWER -> packStartAnswer(packer, body as StartAnswerBody)
            MessageType.ASSISTANT_SENTENCE -> packAssistantSentence(packer, body as AssistantSentenceBody)
            MessageType.CONFIGURATION -> packConfiguration(packer, body as ConfigurationBody)
            MessageType.CONTROL_STOP -> packControlStop(packer, body as ControlStopBody)
            MessageType.ACKNOWLEDGEMENT -> packAcknowledgement(packer, body as AcknowledgementBody)
            MessageType.TOOL_USE_REQUEST -> packToolUseRequest(packer, body as ToolUseRequestBody)
            MessageType.TOOL_USE_RESULT -> packToolUseResult(packer, body as ToolUseResultBody)
            MessageType.REASONING_STEP -> packReasoningStep(packer, body as ReasoningStepBody)
            MessageType.MEMORY_TRACE -> packMemoryTrace(packer, body as MemoryTraceBody)
            MessageType.COMMENTARY -> packCommentary(packer, body as CommentaryBody)
            MessageType.AUDIO_CHUNK -> packAudioChunk(packer, body as AudioChunkBody)
            MessageType.CONTROL_VARIATION -> packControlVariation(packer, body as ControlVariationBody)
        }
    }

    private fun unpackBody(unpacker: MessageUnpacker, type: MessageType): Any {
        return when (type) {
            MessageType.ERROR_MESSAGE -> unpackErrorMessage(unpacker)
            MessageType.USER_MESSAGE -> unpackUserMessage(unpacker)
            MessageType.ASSISTANT_MESSAGE -> unpackAssistantMessage(unpacker)
            MessageType.TRANSCRIPTION -> unpackTranscription(unpacker)
            MessageType.START_ANSWER -> unpackStartAnswer(unpacker)
            MessageType.ASSISTANT_SENTENCE -> unpackAssistantSentence(unpacker)
            MessageType.CONFIGURATION -> unpackConfiguration(unpacker)
            MessageType.CONTROL_STOP -> unpackControlStop(unpacker)
            MessageType.ACKNOWLEDGEMENT -> unpackAcknowledgement(unpacker)
            MessageType.TOOL_USE_REQUEST -> unpackToolUseRequest(unpacker)
            MessageType.TOOL_USE_RESULT -> unpackToolUseResult(unpacker)
            MessageType.REASONING_STEP -> unpackReasoningStep(unpacker)
            MessageType.MEMORY_TRACE -> unpackMemoryTrace(unpacker)
            MessageType.COMMENTARY -> unpackCommentary(unpacker)
            MessageType.AUDIO_CHUNK -> unpackAudioChunk(unpacker)
            MessageType.CONTROL_VARIATION -> unpackControlVariation(unpacker)
        }
    }

    // Helper functions for packing
    private fun packMap(packer: org.msgpack.core.MessagePacker, map: Map<String, Any>) {
        packer.packMapHeader(map.size)
        for ((key, value) in map) {
            packer.packString(key)
            packValue(packer, value)
        }
    }

    /**
     * Pack a value into MessagePack format.
     * Supported types: String, Int, Long, Float, Double, Boolean, ByteArray, Map<String,Any>, List.
     * Unknown types are coerced to String via toString() with a warning logged.
     */
    private fun packValue(packer: org.msgpack.core.MessagePacker, value: Any?) {
        when (value) {
            null -> packer.packNil()
            is String -> packer.packString(value)
            is Int -> packer.packInt(value)
            is Long -> packer.packLong(value)
            is Float -> packer.packFloat(value)
            is Double -> packer.packDouble(value)
            is Boolean -> packer.packBoolean(value)
            is ByteArray -> packer.packBinaryHeader(value.size).writePayload(value)
            is Map<*, *> -> {
                // Validate that all keys are Strings before casting
                if (value.keys.all { it is String }) {
                    @Suppress("UNCHECKED_CAST")
                    packMap(packer, value as Map<String, Any>)
                } else {
                    throw IllegalArgumentException("Map keys must be Strings")
                }
            }
            is List<*> -> {
                packer.packArrayHeader(value.size)
                value.forEach { packValue(packer, it) }
            }
            else -> {
                // Warning: Unknown type coerced to String. Prefer explicit types in protocol definitions.
                Timber.w("packValue: Unknown type ${value::class.simpleName} coerced to String")
                packer.packString(value.toString())
            }
        }
    }

    private fun unpackMap(unpacker: MessageUnpacker): Map<String, Any?> {
        val size = unpacker.unpackMapHeader()
        val map = mutableMapOf<String, Any?>()
        for (i in 0 until size) {
            val key = unpacker.unpackString()
            val value = unpackValue(unpacker)
            map[key] = value
        }
        return map
    }

    private fun unpackValue(unpacker: MessageUnpacker): Any? {
        val format = unpacker.nextFormat
        return when {
            format.valueType == org.msgpack.value.ValueType.NIL -> {
                unpacker.unpackNil()
                null
            }
            format.valueType == org.msgpack.value.ValueType.BOOLEAN -> unpacker.unpackBoolean()
            format.valueType == org.msgpack.value.ValueType.INTEGER -> {
                if (format.valueType.isIntegerType) {
                    when {
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.POSFIXINT ||
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.NEGFIXINT ||
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.INT8 ||
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.INT16 ||
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.INT32 ||
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.UINT8 ||
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.UINT16 ||
                        unpacker.nextFormat == org.msgpack.core.MessageFormat.UINT32 -> unpacker.unpackInt()
                        else -> unpacker.unpackLong()
                    }
                } else {
                    unpacker.unpackLong()
                }
            }
            format.valueType == org.msgpack.value.ValueType.FLOAT -> {
                if (unpacker.nextFormat == org.msgpack.core.MessageFormat.FLOAT32) {
                    unpacker.unpackFloat()
                } else {
                    unpacker.unpackDouble()
                }
            }
            format.valueType == org.msgpack.value.ValueType.STRING -> unpacker.unpackString()
            format.valueType == org.msgpack.value.ValueType.BINARY -> {
                val size = unpacker.unpackBinaryHeader()
                unpacker.readPayload(size)
            }
            format.valueType == org.msgpack.value.ValueType.ARRAY -> {
                val size = unpacker.unpackArrayHeader()
                List(size) { unpackValue(unpacker) }
            }
            format.valueType == org.msgpack.value.ValueType.MAP -> unpackMap(unpacker)
            else -> throw IllegalArgumentException("Unsupported value type: ${format.valueType}")
        }
    }

    // Pack/Unpack methods for each message type
    private fun packTranscription(packer: org.msgpack.core.MessagePacker, body: TranscriptionBody) {
        // Always packs 7 fields: id, previousId, conversationId, text, final, confidence, language
        // Optional fields (previousId, confidence, language) are packed as nil when absent
        packer.packMapHeader(7)
        packer.packString("id").packString(body.id)
        packer.packString("previousId")
        if (body.previousId != null) packer.packString(body.previousId) else packer.packNil()
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("text").packString(body.text)
        packer.packString("final").packBoolean(body.final)
        packer.packString("confidence")
        if (body.confidence != null) packer.packFloat(body.confidence) else packer.packNil()
        packer.packString("language")
        if (body.language != null) packer.packString(body.language) else packer.packNil()
    }

    private fun unpackTranscription(unpacker: MessageUnpacker): TranscriptionBody {
        val size = unpacker.unpackMapHeader()
        var id: String? = null
        var previousId: String? = null
        var conversationId: String? = null
        var text: String? = null
        var final = false
        var confidence: Float? = null
        var language: String? = null

        for (i in 0 until size) {
            when (unpacker.unpackString()) {
                "id" -> id = unpacker.unpackString()
                "previousId" -> previousId = if (unpacker.tryUnpackNil()) null else unpacker.unpackString()
                "conversationId" -> conversationId = unpacker.unpackString()
                "text" -> text = unpacker.unpackString()
                "final" -> final = unpacker.unpackBoolean()
                "confidence" -> confidence = if (unpacker.tryUnpackNil()) null else unpacker.unpackFloat()
                "language" -> language = if (unpacker.tryUnpackNil()) null else unpacker.unpackString()
            }
        }

        return TranscriptionBody(
            id = id ?: throw IllegalArgumentException("Missing id"),
            previousId = previousId,
            conversationId = conversationId ?: throw IllegalArgumentException("Missing conversationId"),
            text = text ?: throw IllegalArgumentException("Missing text"),
            final = final,
            confidence = confidence,
            language = language
        )
    }

    private fun packStartAnswer(packer: org.msgpack.core.MessagePacker, body: StartAnswerBody) {
        // Always packs 5 fields: id, previousId, conversationId, answerType, plannedSentenceCount
        // Optional fields (answerType, plannedSentenceCount) are packed as nil when absent
        packer.packMapHeader(5)
        packer.packString("id").packString(body.id)
        packer.packString("previousId").packString(body.previousId)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("answerType")
        if (body.answerType != null) {
            // Convert enum underscores to plus signs (e.g., TEXT_VOICE -> text+voice)
            packer.packString(body.answerType.name.lowercase().replace("_", "+"))
        } else {
            packer.packNil()
        }
        packer.packString("plannedSentenceCount")
        if (body.plannedSentenceCount != null) packer.packInt(body.plannedSentenceCount) else packer.packNil()
    }

    private fun unpackStartAnswer(unpacker: MessageUnpacker): StartAnswerBody {
        val size = unpacker.unpackMapHeader()
        var id: String? = null
        var previousId: String? = null
        var conversationId: String? = null
        var answerType: AnswerType? = null
        var plannedSentenceCount: Int? = null

        for (i in 0 until size) {
            when (unpacker.unpackString()) {
                "id" -> id = unpacker.unpackString()
                "previousId" -> previousId = unpacker.unpackString()
                "conversationId" -> conversationId = unpacker.unpackString()
                "answerType" -> answerType = if (unpacker.tryUnpackNil()) null else AnswerType.fromString(unpacker.unpackString())
                "plannedSentenceCount" -> plannedSentenceCount = if (unpacker.tryUnpackNil()) null else unpacker.unpackInt()
            }
        }

        return StartAnswerBody(
            id = id ?: throw IllegalArgumentException("Missing id"),
            previousId = previousId ?: throw IllegalArgumentException("Missing previousId"),
            conversationId = conversationId ?: throw IllegalArgumentException("Missing conversationId"),
            answerType = answerType,
            plannedSentenceCount = plannedSentenceCount
        )
    }

    private fun packAssistantSentence(packer: org.msgpack.core.MessagePacker, body: AssistantSentenceBody) {
        // Always packs 7 fields: id, previousId, conversationId, sequence, text, isFinal, audio
        // Optional fields (id, isFinal, audio) are packed as nil when absent
        packer.packMapHeader(7)
        packer.packString("id")
        if (body.id != null) packer.packString(body.id) else packer.packNil()
        packer.packString("previousId").packString(body.previousId)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("sequence").packInt(body.sequence)
        packer.packString("text").packString(body.text)
        packer.packString("isFinal")
        if (body.isFinal != null) packer.packBoolean(body.isFinal) else packer.packNil()
        packer.packString("audio")
        if (body.audio != null) {
            packer.packBinaryHeader(body.audio.size).writePayload(body.audio)
        } else {
            packer.packNil()
        }
    }

    private fun unpackAssistantSentence(unpacker: MessageUnpacker): AssistantSentenceBody {
        val size = unpacker.unpackMapHeader()
        var id: String? = null
        var previousId: String? = null
        var conversationId: String? = null
        var sequence: Int? = null
        var text: String? = null
        var isFinal: Boolean? = null
        var audio: ByteArray? = null

        for (i in 0 until size) {
            when (unpacker.unpackString()) {
                "id" -> id = if (unpacker.tryUnpackNil()) null else unpacker.unpackString()
                "previousId" -> previousId = unpacker.unpackString()
                "conversationId" -> conversationId = unpacker.unpackString()
                "sequence" -> sequence = unpacker.unpackInt()
                "text" -> text = unpacker.unpackString()
                "isFinal" -> isFinal = if (unpacker.tryUnpackNil()) null else unpacker.unpackBoolean()
                "audio" -> audio = if (unpacker.tryUnpackNil()) null else {
                    val audioSize = unpacker.unpackBinaryHeader()
                    unpacker.readPayload(audioSize)
                }
            }
        }

        return AssistantSentenceBody(
            id = id,
            previousId = previousId ?: throw IllegalArgumentException("Missing previousId"),
            conversationId = conversationId ?: throw IllegalArgumentException("Missing conversationId"),
            sequence = sequence ?: throw IllegalArgumentException("Missing sequence"),
            text = text ?: throw IllegalArgumentException("Missing text"),
            isFinal = isFinal,
            audio = audio
        )
    }

    // Message type serialization methods.
    // All types use explicit field packing. Most use generic unpacking (unpackMap),
    // while types with ByteArray fields use explicit unpacking for proper binary handling.
    private fun packUserMessage(packer: org.msgpack.core.MessagePacker, body: UserMessageBody) {
        packer.packMapHeader(5)
        packer.packString("id").packString(body.id)
        packer.packString("previousId")
        if (body.previousId != null) packer.packString(body.previousId) else packer.packNil()
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("content").packString(body.content)
        packer.packString("timestamp")
        if (body.timestamp != null) packer.packLong(body.timestamp) else packer.packNil()
    }

    private fun unpackUserMessage(unpacker: MessageUnpacker): UserMessageBody {
        val map = unpackMap(unpacker)
        return UserMessageBody(
            id = map["id"] as String,
            previousId = map["previousId"] as? String,
            conversationId = map["conversationId"] as String,
            content = map["content"] as String,
            timestamp = map["timestamp"] as? Long
        )
    }

    private fun packAssistantMessage(packer: org.msgpack.core.MessagePacker, body: AssistantMessageBody) {
        packer.packMapHeader(5)
        packer.packString("id").packString(body.id)
        packer.packString("previousId")
        if (body.previousId != null) packer.packString(body.previousId) else packer.packNil()
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("content").packString(body.content)
        packer.packString("timestamp")
        if (body.timestamp != null) packer.packLong(body.timestamp) else packer.packNil()
    }

    private fun unpackAssistantMessage(unpacker: MessageUnpacker): AssistantMessageBody {
        val map = unpackMap(unpacker)
        return AssistantMessageBody(
            id = map["id"] as String,
            previousId = map["previousId"] as? String,
            conversationId = map["conversationId"] as String,
            content = map["content"] as String,
            timestamp = map["timestamp"] as? Long
        )
    }

    private fun packConfiguration(packer: org.msgpack.core.MessagePacker, body: ConfigurationBody) {
        packer.packMapHeader(6)
        packer.packString("conversationId")
        if (body.conversationId != null) packer.packString(body.conversationId) else packer.packNil()
        packer.packString("lastSequenceSeen")
        if (body.lastSequenceSeen != null) packer.packInt(body.lastSequenceSeen) else packer.packNil()
        packer.packString("clientVersion")
        if (body.clientVersion != null) packer.packString(body.clientVersion) else packer.packNil()
        packer.packString("preferredLanguage")
        if (body.preferredLanguage != null) packer.packString(body.preferredLanguage) else packer.packNil()
        packer.packString("device")
        if (body.device != null) packer.packString(body.device) else packer.packNil()
        packer.packString("features")
        if (body.features != null) {
            packer.packArrayHeader(body.features.size)
            body.features.forEach { packer.packString(it) }
        } else {
            packer.packNil()
        }
    }

    private fun unpackConfiguration(unpacker: MessageUnpacker): ConfigurationBody {
        val map = unpackMap(unpacker)
        return ConfigurationBody(
            conversationId = map["conversationId"] as? String,
            lastSequenceSeen = (map["lastSequenceSeen"] as? Number)?.toInt(),
            clientVersion = map["clientVersion"] as? String,
            preferredLanguage = map["preferredLanguage"] as? String,
            device = map["device"] as? String,
            features = (map["features"] as? List<*>)?.filterIsInstance<String>()
        )
    }

    private fun packControlStop(packer: org.msgpack.core.MessagePacker, body: ControlStopBody) {
        packer.packMapHeader(4)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("targetId")
        if (body.targetId != null) packer.packString(body.targetId) else packer.packNil()
        packer.packString("reason")
        if (body.reason != null) packer.packString(body.reason) else packer.packNil()
        packer.packString("stopType")
        if (body.stopType != null) packer.packString(body.stopType.name.lowercase()) else packer.packNil()
    }

    private fun unpackControlStop(unpacker: MessageUnpacker): ControlStopBody {
        val map = unpackMap(unpacker)
        return ControlStopBody(
            conversationId = map["conversationId"] as String,
            targetId = map["targetId"] as? String,
            reason = map["reason"] as? String,
            stopType = StopType.fromString(map["stopType"] as? String)
        )
    }

    private fun packAcknowledgement(packer: org.msgpack.core.MessagePacker, body: AcknowledgementBody) {
        packer.packMapHeader(3)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("acknowledgedStanzaId").packInt(body.acknowledgedStanzaId)
        packer.packString("success").packBoolean(body.success)
    }

    private fun unpackAcknowledgement(unpacker: MessageUnpacker): AcknowledgementBody {
        val map = unpackMap(unpacker)
        return AcknowledgementBody(
            conversationId = map["conversationId"] as String,
            acknowledgedStanzaId = (map["acknowledgedStanzaId"] as Number).toInt(),
            success = map["success"] as Boolean
        )
    }

    private fun packErrorMessage(packer: org.msgpack.core.MessagePacker, body: ErrorMessageBody) {
        packer.packMapHeader(7)
        packer.packString("id").packString(body.id)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("code").packInt(body.code)
        packer.packString("message").packString(body.message)
        packer.packString("severity").packInt(body.severity.value)
        packer.packString("recoverable").packBoolean(body.recoverable)
        packer.packString("originatingId")
        if (body.originatingId != null) packer.packString(body.originatingId) else packer.packNil()
    }

    private fun unpackErrorMessage(unpacker: MessageUnpacker): ErrorMessageBody {
        val map = unpackMap(unpacker)
        return ErrorMessageBody(
            id = map["id"] as String,
            conversationId = map["conversationId"] as String,
            code = (map["code"] as Number).toInt(),
            message = map["message"] as String,
            severity = Severity.fromInt((map["severity"] as Number).toInt()),
            recoverable = map["recoverable"] as Boolean,
            originatingId = map["originatingId"] as? String
        )
    }

    private fun packToolUseRequest(packer: org.msgpack.core.MessagePacker, body: ToolUseRequestBody) {
        packer.packMapHeader(7)
        packer.packString("id").packString(body.id)
        packer.packString("messageId").packString(body.messageId)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("toolName").packString(body.toolName)
        packer.packString("parameters")
        packMap(packer, body.parameters)
        packer.packString("execution").packString(body.execution.name.lowercase())
        packer.packString("timeoutMs")
        if (body.timeoutMs != null) packer.packInt(body.timeoutMs) else packer.packNil()
    }

    private fun unpackToolUseRequest(unpacker: MessageUnpacker): ToolUseRequestBody {
        val map = unpackMap(unpacker)

        // Validate parameters field is a Map before casting
        val parameters = map["parameters"]
        if (parameters !is Map<*, *>) {
            throw IllegalArgumentException("parameters field must be a Map")
        }
        if (!parameters.keys.all { it is String }) {
            throw IllegalArgumentException("parameters Map keys must be Strings")
        }

        @Suppress("UNCHECKED_CAST")
        return ToolUseRequestBody(
            id = map["id"] as String,
            messageId = map["messageId"] as String,
            conversationId = map["conversationId"] as String,
            toolName = map["toolName"] as String,
            parameters = parameters as Map<String, Any>,
            execution = ToolExecution.fromString(map["execution"] as? String) ?: ToolExecution.SERVER,
            timeoutMs = (map["timeoutMs"] as? Number)?.toInt()
        )
    }

    private fun packToolUseResult(packer: org.msgpack.core.MessagePacker, body: ToolUseResultBody) {
        packer.packMapHeader(7)
        packer.packString("id").packString(body.id)
        packer.packString("requestId").packString(body.requestId)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("success").packBoolean(body.success)
        packer.packString("result")
        if (body.result != null) packValue(packer, body.result) else packer.packNil()
        packer.packString("errorCode")
        if (body.errorCode != null) packer.packString(body.errorCode) else packer.packNil()
        packer.packString("errorMessage")
        if (body.errorMessage != null) packer.packString(body.errorMessage) else packer.packNil()
    }

    private fun unpackToolUseResult(unpacker: MessageUnpacker): ToolUseResultBody {
        val map = unpackMap(unpacker)
        return ToolUseResultBody(
            id = map["id"] as String,
            requestId = map["requestId"] as String,
            conversationId = map["conversationId"] as String,
            success = map["success"] as Boolean,
            result = map["result"],
            errorCode = map["errorCode"] as? String,
            errorMessage = map["errorMessage"] as? String
        )
    }

    private fun packReasoningStep(packer: org.msgpack.core.MessagePacker, body: ReasoningStepBody) {
        packer.packMapHeader(5)
        packer.packString("id").packString(body.id)
        packer.packString("messageId").packString(body.messageId)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("sequence").packInt(body.sequence)
        packer.packString("content").packString(body.content)
    }

    private fun unpackReasoningStep(unpacker: MessageUnpacker): ReasoningStepBody {
        val map = unpackMap(unpacker)
        return ReasoningStepBody(
            id = map["id"] as String,
            messageId = map["messageId"] as String,
            conversationId = map["conversationId"] as String,
            sequence = (map["sequence"] as? Number)?.toInt() ?: throw IllegalArgumentException("Missing sequence"),
            content = map["content"] as String
        )
    }

    private fun packMemoryTrace(packer: org.msgpack.core.MessagePacker, body: MemoryTraceBody) {
        packer.packMapHeader(6)
        packer.packString("id").packString(body.id)
        packer.packString("messageId").packString(body.messageId)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("memoryId").packString(body.memoryId)
        packer.packString("content").packString(body.content)
        packer.packString("relevance").packFloat(body.relevance)
    }

    private fun unpackMemoryTrace(unpacker: MessageUnpacker): MemoryTraceBody {
        val map = unpackMap(unpacker)
        return MemoryTraceBody(
            id = map["id"] as String,
            messageId = map["messageId"] as String,
            conversationId = map["conversationId"] as String,
            memoryId = map["memoryId"] as String,
            content = map["content"] as String,
            relevance = (map["relevance"] as? Number)?.toFloat() ?: throw IllegalArgumentException("Missing relevance")
        )
    }

    private fun packCommentary(packer: org.msgpack.core.MessagePacker, body: CommentaryBody) {
        packer.packMapHeader(5)
        packer.packString("id").packString(body.id)
        packer.packString("messageId").packString(body.messageId)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("content").packString(body.content)
        packer.packString("commentaryType")
        if (body.commentaryType != null) packer.packString(body.commentaryType) else packer.packNil()
    }

    private fun unpackCommentary(unpacker: MessageUnpacker): CommentaryBody {
        val map = unpackMap(unpacker)
        return CommentaryBody(
            id = map["id"] as String,
            messageId = map["messageId"] as String,
            conversationId = map["conversationId"] as String,
            content = map["content"] as String,
            commentaryType = map["commentaryType"] as? String
        )
    }

    private fun packAudioChunk(packer: org.msgpack.core.MessagePacker, body: AudioChunkBody) {
        packer.packMapHeader(8)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("format").packString(body.format)
        packer.packString("sequence").packInt(body.sequence)
        packer.packString("durationMs").packInt(body.durationMs)
        packer.packString("trackSid")
        if (body.trackSid != null) packer.packString(body.trackSid) else packer.packNil()
        packer.packString("data")
        if (body.data != null) {
            packer.packBinaryHeader(body.data.size).writePayload(body.data)
        } else {
            packer.packNil()
        }
        packer.packString("isLast")
        if (body.isLast != null) packer.packBoolean(body.isLast) else packer.packNil()
        packer.packString("timestamp")
        if (body.timestamp != null) packer.packLong(body.timestamp) else packer.packNil()
    }

    private fun unpackAudioChunk(unpacker: MessageUnpacker): AudioChunkBody {
        val size = unpacker.unpackMapHeader()
        var conversationId: String? = null
        var format: String? = null
        var sequence: Int? = null
        var durationMs: Int? = null
        var trackSid: String? = null
        var data: ByteArray? = null
        var isLast: Boolean? = null
        var timestamp: Long? = null

        for (i in 0 until size) {
            when (unpacker.unpackString()) {
                "conversationId" -> conversationId = unpacker.unpackString()
                "format" -> format = unpacker.unpackString()
                "sequence" -> sequence = unpacker.unpackInt()
                "durationMs" -> durationMs = unpacker.unpackInt()
                "trackSid" -> trackSid = if (unpacker.tryUnpackNil()) null else unpacker.unpackString()
                "data" -> data = if (unpacker.tryUnpackNil()) null else {
                    val dataSize = unpacker.unpackBinaryHeader()
                    unpacker.readPayload(dataSize)
                }
                "isLast" -> isLast = if (unpacker.tryUnpackNil()) null else unpacker.unpackBoolean()
                "timestamp" -> timestamp = if (unpacker.tryUnpackNil()) null else unpacker.unpackLong()
            }
        }

        return AudioChunkBody(
            conversationId = conversationId ?: throw IllegalArgumentException("Missing conversationId"),
            format = format ?: throw IllegalArgumentException("Missing format"),
            sequence = sequence ?: throw IllegalArgumentException("Missing sequence"),
            durationMs = durationMs ?: throw IllegalArgumentException("Missing durationMs"),
            trackSid = trackSid,
            data = data,
            isLast = isLast,
            timestamp = timestamp
        )
    }

    private fun packControlVariation(packer: org.msgpack.core.MessagePacker, body: ControlVariationBody) {
        packer.packMapHeader(4)
        packer.packString("conversationId").packString(body.conversationId)
        packer.packString("targetId").packString(body.targetId)
        packer.packString("mode").packString(body.mode.name.lowercase())
        packer.packString("newContent")
        if (body.newContent != null) packer.packString(body.newContent) else packer.packNil()
    }

    private fun unpackControlVariation(unpacker: MessageUnpacker): ControlVariationBody {
        val map = unpackMap(unpacker)
        return ControlVariationBody(
            conversationId = map["conversationId"] as String,
            targetId = map["targetId"] as String,
            mode = VariationType.fromString(map["mode"] as? String)
                ?: throw IllegalArgumentException("Invalid mode"),
            newContent = map["newContent"] as? String
        )
    }
}
