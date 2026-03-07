package com.example.plotix_mobile.presentation.main

import com.example.plotix_mobile.domain.model.ChatContact

/**
 * UI State — единственный источник данных для экрана.
 */
data class MainScreenState(
    val contacts: List<ChatContact> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null
)