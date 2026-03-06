package com.example.plotix_mobile.domain.model

import kotlin.time.Clock

data class Message(
    val id: String,
    val text: String,
    val senderId: String,
    val isFromMe: Boolean,
    val timestamp: Long = Clock.System.now().toEpochMilliseconds()
)