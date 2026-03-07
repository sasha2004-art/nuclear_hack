package com.example.plotix_mobile.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import cafe.adriel.voyager.core.model.rememberScreenModel
import cafe.adriel.voyager.core.screen.Screen
import cafe.adriel.voyager.navigator.LocalNavigator
import cafe.adriel.voyager.navigator.currentOrThrow
import com.example.plotix_mobile.domain.getChatRepository
import com.example.plotix_mobile.domain.model.ChatContact
import com.example.plotix_mobile.presentation.chat.ChatScreenModel
import com.example.plotix_mobile.ui.theme.Colors

data class ChatScreen(val contact: ChatContact) : Screen {
    @OptIn(ExperimentalMaterial3Api::class)
    @Composable
    override fun Content() {
        val navigator = LocalNavigator.currentOrThrow
        // Используем ScreenModel для чата
        val screenModel = rememberScreenModel { ChatScreenModel(contact, getChatRepository()) }
        val messages by screenModel.messages.collectAsState()
        val inputText by screenModel.inputText.collectAsState()

        Scaffold(
            containerColor = Colors.Background,
            topBar = {
                TopAppBar(
                    title = { Text(contact.displayName, color = Color.White) },
                    navigationIcon = {
                        IconButton(onClick = { navigator.pop() }) { // Назад
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back", tint = Color.White)
                        }
                    },
                    colors = TopAppBarDefaults.topAppBarColors(containerColor = Colors.Background)
                )
            },
            bottomBar = {
                // Поле ввода сообщения
                Row(modifier = Modifier.padding(8.dp).fillMaxWidth().imePadding(), verticalAlignment = Alignment.CenterVertically) {
                    TextField(
                        value = inputText,
                        onValueChange = { screenModel.onTextChanged(it) },
                        modifier = Modifier.weight(1f),
                        placeholder = { Text("Сообщение...", color = Color.Gray) },
                        colors = TextFieldDefaults.colors(
                            focusedContainerColor = Color(0xFF1E232E),
                            unfocusedContainerColor = Color(0xFF1E232E),
                            focusedIndicatorColor = Color.Transparent,
                            unfocusedIndicatorColor = Color.Transparent,
                            focusedTextColor = Color.White
                        )
                    )
                    IconButton(onClick = { screenModel.sendMessage() }) {
                        Icon(Icons.AutoMirrored.Filled.Send, "Send", tint = Colors.Accent)
                    }
                }
            }
        ) { padding ->
            LazyColumn(modifier = Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp)) {
                items(messages) { msg ->
                    // Простая верстка пузырька сообщения
                    Box(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
                        contentAlignment = if (msg.isFromMe) Alignment.CenterEnd else Alignment.CenterStart) {
                        Surface(color = if (msg.isFromMe) Colors.Accent else Color(0xFF1E232E), shape = MaterialTheme.shapes.medium) {
                            Text(msg.text, modifier = Modifier.padding(12.dp), color = Color.White)
                        }
                    }
                }
            }
        }
    }
}
