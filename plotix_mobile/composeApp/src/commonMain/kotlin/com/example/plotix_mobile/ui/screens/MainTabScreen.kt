package com.example.plotix_mobile.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import cafe.adriel.voyager.core.model.rememberScreenModel
import cafe.adriel.voyager.core.screen.Screen
import cafe.adriel.voyager.navigator.LocalNavigator
import cafe.adriel.voyager.navigator.currentOrThrow
import com.example.plotix_mobile.presentation.main.MainScreenModel
import com.example.plotix_mobile.ui.components.ChatListItem
import com.example.plotix_mobile.ui.theme.Colors

class MainTabScreen : Screen {
    @OptIn(ExperimentalMaterial3Api::class)
    @Composable
    override fun Content() {
        val navigator = LocalNavigator.currentOrThrow
        val screenModel = rememberScreenModel { MainScreenModel() }
        val state by screenModel.state.collectAsState()

        Scaffold(
            containerColor = Colors.Background,
            topBar = {
                TopAppBar(
                    title = { Text("Plotix Local", color = Colors.Accent, fontWeight = FontWeight.Bold) },
                    actions = {
                        IconButton(onClick = {}) { Icon(Icons.Default.Search, "Search", tint = Color.White) }
                        IconButton(onClick = {}) { Icon(Icons.Default.Settings, "Settings", tint = Color.White) }
                    },
                    colors = TopAppBarDefaults.topAppBarColors(containerColor = Colors.Background)
                )
            }
        ) { padding ->
            LazyColumn(modifier = Modifier.fillMaxSize().padding(padding)) {
                items(state.chats, key = { it.id }) { chat ->
                    ChatListItem(
                        chat = chat,
                        isSelected = false,
                        onClick = { navigator.push(ChatScreen(chat)) } // Навигация работает!
                    )
                }
            }
        }
    }
}