package com.example.plotix_mobile

import androidx.compose.ui.window.ComposeUIViewController
import com.example.plotix_mobile.presentation.main.MainViewModel
import com.example.plotix_mobile.ui.screens.MainScreen

fun MainViewController() = ComposeUIViewController {
    val viewModel = MainViewModel()

    MainScreen(viewModel = viewModel)
}