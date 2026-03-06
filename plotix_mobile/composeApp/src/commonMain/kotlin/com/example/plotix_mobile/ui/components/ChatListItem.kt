package com.example.plotix_mobile.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.plotix_mobile.ui.theme.Colors
import com.example.plotix_mobile.domain.model.ChatContact

@Composable
fun ChatListItem(
    chat: ChatContact,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    // используем Box для наложения индикатора выбора
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(72.dp)
            .background(if (isSelected) Colors.Surface else Color.Transparent)
            .clickable { onClick() }
    ) {
        // Левый индикатор активного чата (как на скриншоте)
        if (isSelected) {
            Box(
                modifier = Modifier
                    .width(4.dp)
                    .fillMaxHeight(0.5f)
                    .align(Alignment.CenterStart)
                    .clip(CircleShape)
                    .background(Colors.SelectionSideBar)
            )
        }

        Row(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Аватар с индикатором статуса
            Box {
                Box(
                    modifier = Modifier
                        .size(48.dp)
                        .clip(CircleShape)
                        .background(Color(0xFF35393E)),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        chat.avatarLetter,
                        color = Color.White,
                        fontSize = 18.sp,
                        fontWeight = FontWeight.Medium
                    )
                }

                // Статус точка
                Box(
                    modifier = Modifier
                        .size(14.dp)
                        .align(Alignment.BottomEnd)
                        .clip(CircleShape)
                        .background(Colors.Background) // Обводка цветом фона
                        .padding(2.dp)
                ) {
                    Box(
                        modifier = Modifier
                            .fillMaxSize()
                            .clip(CircleShape)
                            .background(if (chat.isOnline) Colors.OnlineGreen else Colors.TextSecondary)
                    )
                }
            }

            Spacer(Modifier.width(16.dp))

            Column {
                Text(
                    text = chat.displayName,
                    color = Colors.TextPrimary,
                    fontSize = 16.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1
                )
                Text(
                    text = chat.lastSeen,
                    color = Colors.TextSecondary,
                    fontSize = 13.sp
                )
            }
        }
    }
}