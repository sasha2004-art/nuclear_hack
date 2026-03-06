package com.example.plotix_mobile.domain.model

/**
 * Сущность чата. Используем стабильные типы для оптимизации Compose.
 */
data class ChatContact(
    val id: String,
    val displayName: String,
    val lastSeen: String = "Offline",
    val isOnline: Boolean = false,
    val avatarLetter: String = displayName.take(1).uppercase()
)