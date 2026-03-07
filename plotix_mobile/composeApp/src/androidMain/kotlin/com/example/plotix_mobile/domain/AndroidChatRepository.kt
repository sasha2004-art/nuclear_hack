package com.example.plotix_mobile.domain

import android.util.Log
import com.example.plotix_mobile.domain.model.ChatContact
import com.example.plotix_mobile.domain.model.Message
import io.ktor.client.*
import io.ktor.client.plugins.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.websocket.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.KotlinxWebsocketSerializationConverter
import io.ktor.serialization.kotlinx.json.*
import io.ktor.websocket.readText
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.*
import plotix.MessageListener
import plotix.Plotix

@Serializable
data class WSEvent(
    val type: String,
    val payload: JsonElement
)

class AndroidChatRepository : ChatRepository, MessageListener {
    private val host = "localhost"
    private val port = 8080

    private val client = HttpClient {
        install(ContentNegotiation) {
            json(Json {
                ignoreUnknownKeys = true
                prettyPrint = true
                isLenient = true
            })
        }
        install(WebSockets) {
            contentConverter = KotlinxWebsocketSerializationConverter(Json {
                ignoreUnknownKeys = true
            })
        }
        install(HttpTimeout) {
            requestTimeoutMillis = 5000
            connectTimeoutMillis = 5000
            socketTimeoutMillis = 5000
        }
    }
    private val json = Json { ignoreUnknownKeys = true }
    private val repositoryScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _incomingMessages = MutableSharedFlow<Message>(extraBufferCapacity = 50)
    override val incomingMessages: SharedFlow<Message> = _incomingMessages.asSharedFlow()

    private val _peerUpdates = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    override val peerUpdates: SharedFlow<Unit> = _peerUpdates.asSharedFlow()

    private var isServerReady = false

    init {
        Plotix.registerListener(this)
        connectToWebSocket()
    }

    private fun connectToWebSocket() {
        repositoryScope.launch {
            while (isActive) {
                try {
                    client.webSocket(method = HttpMethod.Get, host = host, port = port, path = "/events") {
                        Log.d("ChatRepo", "WebSocket connected - Server is READY")
                        isServerReady = true
                        for (frame in incoming) {
                            if (frame is io.ktor.websocket.Frame.Text) {
                                val text = frame.readText()
                                try {
                                    val event = json.decodeFromString<WSEvent>(text)
                                    when (event.type) {
                                        "new_message" -> {
                                            val payload = event.payload.jsonObject
                                            val now = System.currentTimeMillis()
                                            val msg = Message(
                                                id = payload["id"]?.jsonPrimitive?.content ?: "msg_$now",
                                                text = payload["text"]?.jsonPrimitive?.content ?: "",
                                                senderId = payload["from"]?.jsonPrimitive?.content ?: "unknown",
                                                isFromMe = false,
                                                timestamp = payload["timestamp"]?.jsonPrimitive?.longOrNull ?: now
                                            )
                                            _incomingMessages.emit(msg)
                                        }
                                        "peer_online", "peer_offline" -> {
                                            _peerUpdates.emit(Unit)
                                        }
                                    }
                                } catch (e: Exception) {
                                    Log.e("ChatRepo", "WS Parse error: ${e.message}")
                                }
                            }
                        }
                    }
                } catch (e: Exception) {
                    isServerReady = false
                    // Silent retry during the first few seconds of startup
                    delay(2000)
                }
            }
        }
    }

    override fun onNewMessage(peerID: String?, text: String?) {
        val now = System.currentTimeMillis()
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
        // Try to fetch with a few retries if server is still starting
        repeat(3) { attempt ->
            try {
                val response = client.get("http://$host:$port/peers").bodyAsText()
                if (response.isNotBlank() && response != "null") {
                    val jsonElement = json.parseToJsonElement(response)
                    return parsePeers(jsonElement)
                }
            } catch (e: Exception) {
                if (attempt < 2) {
                    Log.d("ChatRepo", "Waiting for server to start... (attempt ${attempt + 1})")
                    delay(1000)
                } else {
                    Log.w("ChatRepo", "Could not fetch peers: ${e.message}")
                }
            }
        }
        return emptyList()
    }

    private fun parsePeers(jsonElement: JsonElement): List<ChatContact> {
        return try {
            if (jsonElement is JsonObject) {
                jsonElement.entries.map { (id, element) ->
                    val peer = element.jsonObject
                    ChatContact(
                        id = id,
                        displayName = peer["name"]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() } ?: "Peer ${id.take(8)}",
                        isOnline = peer["online"]?.jsonPrimitive?.booleanOrNull ?: true
                    )
                }
            } else if (jsonElement is JsonArray) {
                jsonElement.map { element ->
                    val peer = element.jsonObject
                    val id = peer["id"]?.jsonPrimitive?.content ?: "unknown"
                    ChatContact(
                        id = id,
                        displayName = peer["name"]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() } ?: "Peer ${id.take(8)}",
                        isOnline = peer["online"]?.jsonPrimitive?.booleanOrNull ?: true
                    )
                }
            } else emptyList()
        } catch (e: Exception) {
            Log.e("ChatRepo", "Parse peers error: ${e.message}")
            emptyList()
        }
    }

    override suspend fun sendMessage(peerId: String, text: String): Result<Unit> {
        return try {
            client.post("http://$host:$port/send_message") {
                contentType(ContentType.Application.Json)
                setBody(buildJsonObject {
                    put("peer_id", peerId)
                    put("text", text)
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
