package com.example.plotix_mobile.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import com.example.plotix_mobile.presentation.main.MainViewModel
import com.example.plotix_mobile.ui.components.ChatListItem
import com.example.plotix_mobile.ui.theme.Colors

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(viewModel: MainViewModel) {
    val state by viewModel.state.collectAsState()

    Scaffold(
        containerColor = Colors.Background,
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        "Plotix Local",
                        color = Colors.Accent,
                        fontWeight = FontWeight.Bold
                    )
                },
                actions = {
                    IconButton(onClick = {}) {
                        Icon(Icons.Default.Search, "Search", tint = Color.White)
                    }
                    IconButton(onClick = {}) {
                        Icon(Icons.Default.Settings, "Settings", tint = Color.White)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Colors.Background
                )
            )
        }
    ) { padding ->
        // Основной список чатов
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            items(
                items = state.chats,
                key = { it.id } // Ключ важен для оптимизации рендеринга списка
            ) { chat ->
                ChatListItem(
                    chat = chat,
                    isSelected = chat.id == state.selectedChatId,
                    onClick = { viewModel.onChatSelected(chat.id) }
                )
            }
        }

        // Добавь здесь UI для Loading или Empty state если нужно
    }
}