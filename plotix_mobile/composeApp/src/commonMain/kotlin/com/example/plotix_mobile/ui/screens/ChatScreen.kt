package com.example.plotix_mobile.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.plotix_mobile.presentation.chat.ChatViewModel
import com.example.plotix_mobile.ui.theme.Colors

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreen(viewModel: ChatViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsState()

    Scaffold(
        containerColor = Colors.Background,
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(state.contact?.displayName ?: "", color = Color.White, fontSize = 16.sp, fontWeight = FontWeight.Bold)
                        Text("OFFLINE", color = Color.Gray, fontSize = 11.sp)
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = Color.White)
                    }
                },
                actions = {
                    IconButton(onClick = {}) {
                        Icon(Icons.Default.MoreVert, contentDescription = "More", tint = Color.White)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = Colors.Background)
            )
        },
        bottomBar = {
            ChatInputBar(
                text = state.inputText,
                onTextChange = viewModel::onTextChanged,
                onSend = viewModel::sendMessage
            )
        }
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(16.dp),
            reverseLayout = false // В реальном чате обычно true, но для начала так проще
        ) {
            items(state.messages) { message ->
                MessageBubble(message)
            }
        }
    }
}

@Composable
fun ChatInputBar(text: String, onTextChange: (String) -> Unit, onSend: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .navigationBarsPadding() // Важно для iOS/Android отступов
            .imePadding() // Поднимает бар при открытии клавиатуры
            .padding(horizontal = 8.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Кнопка вложения
        Surface(
            color = Color(0xFF1E232E),
            shape = RoundedCornerShape(12.dp),
            modifier = Modifier.size(48.dp)
        ) {
            IconButton(onClick = {}) {
                Icon(Icons.Default.AttachFile, null, tint = Color.LightGray)
            }
        }

        Spacer(Modifier.width(8.dp))

        // Поле ввода
        TextField(
            value = text,
            onValueChange = onTextChange,
            placeholder = { Text("Введите сообщение...", color = Color.Gray) },
            modifier = Modifier.weight(1f),
            colors = TextFieldDefaults.colors(
                focusedContainerColor = Color(0xFF1E232E),
                unfocusedContainerColor = Color(0xFF1E232E),
                focusedIndicatorColor = Color.Transparent,
                unfocusedIndicatorColor = Color.Transparent,
                cursorColor = Colors.Accent,
                focusedTextColor = Color.White
            ),
            shape = RoundedCornerShape(12.dp)
        )

        Spacer(Modifier.width(8.dp))

        // Кнопка отправки
        Surface(
            color = Color(0xFF1E232E),
            shape = RoundedCornerShape(12.dp),
            modifier = Modifier.size(48.dp)
        ) {
            IconButton(onClick = onSend) {
                Icon(Icons.AutoMirrored.Filled.Send, null, tint = Colors.Accent)
            }
        }
    }
}

@Composable
fun MessageBubble(message: com.example.plotix_mobile.domain.model.Message) {
    // Здесь будет верстка сообщения, для начала просто текст
    Column(
        modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
        horizontalAlignment = if (message.isFromMe) Alignment.End else Alignment.Start
    ) {
        Surface(
            color = if (message.isFromMe) Colors.Accent else Color(0xFF1E232E),
            shape = RoundedCornerShape(12.dp)
        ) {
            Text(
                message.text,
                modifier = Modifier.padding(12.dp),
                color = Color.White
            )
        }
    }
}