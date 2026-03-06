package com.example.plotix_mobile.presentation.chat

import androidx.lifecycle.ViewModel
import com.example.plotix_mobile.domain.model.Message
import com.example.plotix_mobile.domain.model.ChatContact
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlin.time.Clock

data class ChatUiState(
    val contact: ChatContact? = null,
    val messages: List<Message> = emptyList(),
    val inputText: String = ""
)

class ChatViewModel(val contact: ChatContact) : ViewModel() {
    private val _state = MutableStateFlow(ChatUiState(contact = contact))
    val state = _state.asStateFlow()

    fun onTextChanged(text: String) {
        _state.update { it.copy(inputText = text) }
    }

    fun sendMessage() {
        val text = _state.value.inputText
        if (text.isBlank()) return

        val newMessage = Message(
            id = Clock.System.now().toEpochMilliseconds().toString(),
            text = text,
            senderId = "me",
            isFromMe = true
        )

        _state.update {
            it.copy(
                messages = it.messages + newMessage,
                inputText = "" // Очищаем поле ввода
            )
        }
    }
}