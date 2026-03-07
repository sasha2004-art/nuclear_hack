package com.example.plotix_mobile.domain

import com.example.plotix_mobile.domain.model.ChatContact
import com.example.plotix_mobile.domain.model.Message
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.emptyFlow

class IosChatRepository : ChatRepository {
    override val incomingMessages: Flow<Message> = emptyFlow()
    override val peerUpdates: Flow<Unit> = emptyFlow()

    override suspend fun fetchPeers(): List<ChatContact> = emptyList()
    override suspend fun sendMessage(peerId: String, text: String): Result<Unit> =
        Result.failure(Exception("Not implemented on iOS"))
}

private val repository = IosChatRepository()
actual fun getChatRepository(): ChatRepository = repository
