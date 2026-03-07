package com.example.plotix_mobile

import android.Manifest
import android.content.Context
import android.net.ConnectivityManager
import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import plotix.Plotix

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Запрашиваем разрешения
        requestPermissions()

        setContent { App() }
    }

    private fun requestPermissions() {
        val permissions = mutableListOf(
            Manifest.permission.ACCESS_FINE_LOCATION,
            Manifest.permission.ACCESS_COARSE_LOCATION,
            Manifest.permission.INTERNET
        )

        // Для Android 13+ добавляем NEARBY_WIFI_DEVICES
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.TIRAMISU) {
            permissions.add(Manifest.permission.NEARBY_WIFI_DEVICES)
        }

        val requestPermissionLauncher = registerForActivityResult(
            ActivityResultContracts.RequestMultiplePermissions()
        ) { results ->
            // Проверяем, даны ли сетевые разрешения
            val granted = results.values.all { it }
            if (granted) {
                // Только ПОСЛЕ получения разрешений запускаем ядро
                startGoCore()
            } else {
                println("Разрешения отклонены. Ядро может не найти сеть.")
                // Все равно пробуем, вдруг повезет
                startGoCore()
            }
        }

        requestPermissionLauncher.launch(permissions.toTypedArray())
    }

    private fun getActiveInterface(context: Context): String {
        try {
            val connectivityManager = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
            val activeNetwork = connectivityManager.activeNetwork
            val linkProperties = connectivityManager.getLinkProperties(activeNetwork)

            val name = linkProperties?.interfaceName // вернет "wlan0", "eth0" и т.д.
            Log.d("PLOTIX", "Detected interface: $name")
            return name ?: ""
        } catch (e: Exception) {
            Log.e("PLOTIX", "Failed to get interface: ${e.message}")
            return ""
        }
    }

    private fun startGoCore() {
        Thread {
            try {
                val filesPath = applicationContext.filesDir.absolutePath
                val iface = getActiveInterface(applicationContext)

                // Если мы на эмуляторе и iface пустой, можно попробовать форсировать "eth0"
                // Но лучше передавать то, что нашел ConnectivityManager
                Plotix.start(filesPath, iface)
            } catch (e: Exception) {
                Log.e("PLOTIX", "Core execution failed: ${e.message}")
            }
        }.start()
    }
}