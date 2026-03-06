package com.example.plotix_mobile

import androidx.compose.runtime.Composable
import cafe.adriel.voyager.navigator.Navigator
import cafe.adriel.voyager.transitions.SlideTransition
import com.example.plotix_mobile.ui.screens.MainTabScreen

@Composable
fun App() {
    // Navigator — сердце Voyager. MainTabScreen будет первым экраном.
    Navigator(MainTabScreen()) { navigator ->
        // SlideTransition дает плавный "выезд" экрана справа (как в Telegram)
        SlideTransition(navigator)
    }
}