package com.example.plotix_mobile.presentation.main

import com.example.plotix_mobile.domain.model.ChatContact

/**
 * UI State — единственный источник данных для экрана.
 */
data class MainScreenState(
    val isLoading: Boolean = false,
    val chats: List<ChatContact> = emptyList(),
    val selectedChatId: String? = null,
    val searchQuery: String = ""
)