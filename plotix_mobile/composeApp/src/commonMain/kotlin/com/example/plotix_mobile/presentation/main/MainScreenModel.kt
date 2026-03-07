package com.example.plotix_mobile.presentation.main

import cafe.adriel.voyager.core.model.ScreenModel
import cafe.adriel.voyager.core.model.screenModelScope
import com.example.plotix_mobile.domain.ChatRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

class MainScreenModel(
    private val repository: ChatRepository
) : ScreenModel {

    private val _state = MutableStateFlow(MainScreenState())
    val state = _state.asStateFlow()

    init {
        loadChats()
        observePeerUpdates()
    }

    private fun loadChats() {
        screenModelScope.launch {
            _state.update { it.copy(isLoading = true) }
            try {
                val peers = repository.fetchPeers()
                _state.update {
                    it.copy(
                        contacts = peers,
                        isLoading = false,
                        error = null
                    )
                }
            } catch (e: Exception) {
                _state.update {
                    it.copy(
                        isLoading = false,
                        error = e.message ?: "Unknown error"
                    )
                }
            }
        }
    }

    private fun observePeerUpdates() {
        screenModelScope.launch {
            repository.peerUpdates.collectLatest {
                loadChats()
            }
        }
    }

    fun onChatSelected(chatId: String) {
        // Handle chat selection if needed
    }
}
