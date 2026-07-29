package com.maxtunnel.client

import android.app.Application
import android.content.Context
import android.util.Log
import com.wireguard.android.backend.GoBackend
import com.wireguard.android.backend.Tunnel
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

class MaxTunnelApplication : Application() {
    @Volatile
    private var backendInstance: GoBackend? = null

    val backend: GoBackend
        get() = getBackend(this)

    override fun onCreate() {
        super.onCreate()
        DeployManager.init(this)
        
        
        
        
        CoroutineScope(SupervisorJob() + Dispatchers.IO).launch {
            runCatching {
                val backend = getBackend(this@MaxTunnelApplication)
                val tunnel = WireGuardHelper.WgTunnel()
                backend.setState(tunnel, Tunnel.State.DOWN, null)
                Log.d("MaxTunnelApp", "Успешно очищен фантомный VPN при холодном старте")
            }.onFailure {
                Log.w("MaxTunnelApp", "Не удалось очистить фантомный VPN: ${it.message}")
            }
        }

        
        CoroutineScope(SupervisorJob() + Dispatchers.Main).launch {
            try {
                TunnelManager.running.collect {
                    VpnWidgetProvider.updateAllWidgets(this@MaxTunnelApplication)
                }
            } catch (e: Exception) {
                Log.e("MaxTunnelApp", "Не удалось обновить виджеты: ${e.message}")
            }
        }

        
        val settingsStore = SettingsStore(this)
        CoroutineScope(SupervisorJob() + Dispatchers.Main).launch {
            try {
                settingsStore.loggingEnabled.collect { enabled ->
                    TunnelManager.isLoggingEnabled = enabled
                }
            } catch (e: Exception) {
                Log.e("MaxTunnelApp", "Не удалось отслеживать флаг логирования: ${e.message}")
            }
        }
    }

    fun getBackend(context: Context): GoBackend {
        return backendInstance ?: synchronized(this) {
            backendInstance ?: GoBackend(context.applicationContext).also { backendInstance = it }
        }
    }
}
