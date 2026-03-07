package com.example.plotix_mobile.presentation.chat

import cafe.adriel.voyager.core.model.ScreenModel
import cafe.adriel.voyager.core.model.screenModelScope
import com.example.plotix_mobile.domain.ChatRepository
import com.example.plotix_mobile.domain.model.ChatContact
import com.example.plotix_mobile.domain.model.Message
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock

class ChatScreenModel(
    val contact: ChatContact,
    private val repository: ChatRepository
) : ScreenModel {
    private val _messages = MutableStateFlow<List<Message>>(emptyList())
    val messages = _messages.asStateFlow()

    private val _inputText = MutableStateFlow("")
    val inputText = _inputText.asStateFlow()

    init {
        observeMessages()
    }

    private fun observeMessages() {
        screenModelScope.launch {
            repository.incomingMessages.collectLatest { message ->
                if (message.senderId == contact.id) {
                    _messages.value += message
                }
            }
        }
    }

    fun onTextChanged(text: String) {
        _inputText.value = text
    }

    fun sendMessage() {
        val text = _inputText.value
        if (text.isBlank()) return

        val now = Clock.System.now().toEpochMilliseconds()
        val newMessage = Message(
            id = "me_$now",
            text = text,
            senderId = "me", // Тут должен быть ваш ID
            isFromMe = true,
            timestamp = now
        )

        _messages.value += newMessage
        _inputText.value = ""

        screenModelScope.launch {
            repository.sendMessage(contact.id, text)
        }
    }
}
