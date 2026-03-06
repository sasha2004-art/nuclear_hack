package com.example.plotix_mobile.presentation.main

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.plotix_mobile.domain.model.ChatContact
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

class MainViewModel : ViewModel() {
    private val _state = MutableStateFlow(MainScreenState())
    val state = _state.asStateFlow()

    init {
        loadChats()
    }

    private fun loadChats() {
        viewModelScope.launch {
            _state.update { it.copy(isLoading = true) }

            // Имитация загрузки данных (в будущем здесь будет Repository)
            val mockData = listOf(
                ChatContact("1", "hex_6e994a1ab44b..."),
                ChatContact("2", "Мессенджер секкс"),
                ChatContact("3", "hex_73cd454bf206..."),
                ChatContact("4", "hex_7f5cb3d1ce7f..."),
                ChatContact("5", "hex_d2212e56a93a52fb", isOnline = true)
            )

            _state.update { it.copy(chats = mockData, isLoading = false, selectedChatId = "5") }
        }
    }

    fun onChatSelected(chatId: String) {
        _state.update { it.copy(selectedChatId = chatId) }
    }
}