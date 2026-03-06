package com.example.plotix_mobile.presentation.chat

import cafe.adriel.voyager.core.model.ScreenModel
import com.example.plotix_mobile.domain.model.ChatContact
import com.example.plotix_mobile.domain.model.Message
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlin.time.Clock

class ChatScreenModel(val contact: ChatContact) : ScreenModel {
    private val _messages = MutableStateFlow<List<Message>>(emptyList())
    val messages = _messages.asStateFlow()

    private val _inputText = MutableStateFlow("")
    val inputText = _inputText.asStateFlow()

    fun onTextChanged(text: String) { _inputText.value = text }

    fun sendMessage() {
        if (_inputText.value.isBlank()) return
        val newMessage = Message(
            id = Clock.System.now().toEpochMilliseconds().toString(),
            text = _inputText.value,
            isFromMe = true,
            timestamp = Clock.System.now().toEpochMilliseconds(),
            senderId = "52"
        )
        _messages.value += newMessage
        _inputText.value = ""
    }
}