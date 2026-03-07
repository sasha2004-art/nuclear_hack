package com.example.plotix_mobile.domain

import com.example.plotix_mobile.domain.model.ChatContact
import com.example.plotix_mobile.domain.model.Message
import kotlinx.coroutines.flow.Flow

interface ChatRepository {
    val incomingMessages: Flow<Message>
    val peerUpdates: Flow<Unit>

    suspend fun fetchPeers(): List<ChatContact>
    suspend fun sendMessage(peerId: String, text: String): Result<Unit>
}

expect fun getChatRepository(): ChatRepository
