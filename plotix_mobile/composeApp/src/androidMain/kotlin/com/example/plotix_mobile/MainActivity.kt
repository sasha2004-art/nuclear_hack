package com.example.plotix_mobile

import android.Manifest
import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.net.wifi.WifiManager
import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import plotix.Plotix
import java.net.NetworkInterface

class MainActivity : ComponentActivity() {
    
    private var multicastLock: WifiManager.MulticastLock? = null

    companion object {
        @Volatile
        private var isCoreStarted = false
    }

    private val requestPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { results ->
        if (results.values.all { it }) {
            Log.d("PLOTIX", "All permissions granted")
            startGoCore()
        } else {
            Log.w("PLOTIX", "Some permissions denied, starting core anyway")
            startGoCore()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        acquireMulticastLock()
        checkAndRequestPermissions()
        setContent { App() }
    }

    private fun acquireMulticastLock() {
        try {
            val wifi = applicationContext.getSystemService(Context.WIFI_SERVICE) as WifiManager
            multicastLock = wifi.createMulticastLock("plotix_multicast_lock")
            multicastLock?.setReferenceCounted(true)
            multicastLock?.acquire()
            Log.d("PLOTIX", "Multicast lock acquired")
        } catch (e: Exception) {
            Log.e("PLOTIX", "Failed to acquire multicast lock: ${e.message}")
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        multicastLock?.release()
    }

    private fun checkAndRequestPermissions() {
        val permissions = mutableListOf(
            Manifest.permission.ACCESS_FINE_LOCATION,
            Manifest.permission.ACCESS_COARSE_LOCATION,
            Manifest.permission.INTERNET,
            Manifest.permission.ACCESS_WIFI_STATE,
            Manifest.permission.CHANGE_WIFI_MULTICAST_STATE
        )

        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.TIRAMISU) {
            permissions.add(Manifest.permission.NEARBY_WIFI_DEVICES)
        }

        requestPermissionLauncher.launch(permissions.toTypedArray())
    }

    private fun getActiveInterfaceName(): String {
        return try {
            val connectivityManager = getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
            val activeNetwork = connectivityManager.activeNetwork
            val linkProperties = connectivityManager.getLinkProperties(activeNetwork)
            val name = linkProperties?.interfaceName
            
            if (name != null) {
                Log.d("PLOTIX", "Using active network interface: $name")
                return name
            }

            // Fallback: ищем любой wlan интерфейс
            val interfaces = NetworkInterface.getNetworkInterfaces().toList()
            val wlan = interfaces.find { it.name.startsWith("wlan") && it.isUp }
            wlan?.name ?: "wlan0"
        } catch (e: Exception) {
            Log.e("PLOTIX", "Failed to detect interface: ${e.message}")
            "wlan0"
        }
    }

    private fun startGoCore() {
        if (isCoreStarted) return
        
        Thread {
            synchronized(this) {
                if (isCoreStarted) return@Thread
                try {
                    val filesPath = applicationContext.filesDir.absolutePath
                    val iface = getActiveInterfaceName()
                    
                    Log.i("PLOTIX", "Starting core on interface: $iface")
                    Plotix.start(filesPath, iface)
                    isCoreStarted = true
                } catch (e: Exception) {
                    Log.e("PLOTIX", "Core execution failed: ${e.message}")
                }
            }
        }.start()
    }
}
