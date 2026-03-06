package com.example.plotix_mobile

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.lifecycle.viewmodel.compose.viewModel
import com.example.plotix_mobile.presentation.main.MainViewModel
import com.example.plotix_mobile.ui.screens.MainScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)

        setContent {
            // что экземпляр VM сохранится при повороте экрана и будет очищен при закрытии Activity.
            val mainViewModel: MainViewModel = viewModel { MainViewModel() }

            // Запускаем напрямую главный экран
            MainScreen(viewModel = mainViewModel)
        }
    }
}