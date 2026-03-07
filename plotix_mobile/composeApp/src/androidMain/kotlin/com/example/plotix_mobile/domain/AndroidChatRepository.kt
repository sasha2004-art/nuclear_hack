package com.example.plotix_mobile.domain

import android.util.Log
import com.example.plotix_mobile.domain.model.ChatContact
import com.example.plotix_mobile.domain.model.Message
import io.ktor.client.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.serialization.json.*
import plotix.MessageListener
import plotix.Plotix
import kotlin.time.Clock

class AndroidChatRepository : ChatRepository, MessageListener {
    private val client = HttpClient {
        install(ContentNegotiation) {
            json(Json {
                ignoreUnknownKeys = true
                prettyPrint = true
                isLenient = true
            })
        }
    }
    private val json = Json { ignoreUnknownKeys = true }

    private val _incomingMessages = MutableSharedFlow<Message>(extraBufferCapacity = 50)
    override val incomingMessages: SharedFlow<Message> = _incomingMessages.asSharedFlow()

    private val _peerUpdates = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    override val peerUpdates: SharedFlow<Unit> = _peerUpdates.asSharedFlow()

    init {
        Plotix.registerListener(this)
    }

    override fun onNewMessage(peerID: String?, text: String?) {
        val now = Clock.System.now().toEpochMilliseconds()
        _incomingMessages.tryEmit(
            Message(
                id = "msg_$now",
                text = text ?: "",
                senderId = peerID ?: "unknown",
                isFromMe = false,
                timestamp = now
            )
        )
    }

    override suspend fun fetchPeers(): List<ChatContact> {
        return try {
            // Используем 127.0.0.1 если core запущен на том же устройстве
            val response = client.get("http://127.0.0.1:8080/peers").bodyAsText()
            Log.d("ChatRepo", "Raw Response: $response")
            
            if (response.isBlank()) return emptyList()

            val jsonElement = json.parseToJsonElement(response)
            
            // Если пришел пустой объект {}
            if (jsonElement is JsonObject && jsonElement.isEmpty()) {
                 return emptyList()
            }

            val jsonObject = jsonElement.jsonObject
            jsonObject.entries.map { (id, element) ->
                val peer = element.jsonObject
                ChatContact(
                    id = id,
                    displayName = peer["name"]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() } ?: "Peer ${id.take(8)}",
                    isOnline = peer["online"]?.jsonPrimitive?.booleanOrNull ?: true
                )
            }
        } catch (e: Exception) {
            Log.e("ChatRepo", "Fetch error: ${e.message}", e)
            emptyList()
        }
    }

    override suspend fun sendMessage(peerId: String, text: String): Result<Unit> {
        return try {
            client.post("http://127.0.0.1:8080/send_message") {
                contentType(ContentType.Application.Json)
                setBody(buildJsonObject {
                    put("peer_id", peerId)
                    put("message", text)
                }.toString())
            }
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

private val repository = AndroidChatRepository()
actual fun getChatRepository(): ChatRepository = repository
