package com.maxtunnel.client

import android.content.Context
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import java.io.File
import java.io.FileOutputStream
import java.net.HttpURLConnection
import java.net.URL
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

object DeployManager {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    val isDeploying = MutableStateFlow(false)
    val isUpdating = MutableStateFlow(false)
    val deployProgress = MutableStateFlow(0f)
    val currentStep = MutableStateFlow("")
    val lastResult = MutableStateFlow("")
    val availableVersion = MutableStateFlow("")
    val currentVersion = MutableStateFlow("")

    private const val GITHUB_REPO = "elizqmill/wdtt-server"

    @Volatile
    var activeSession: com.jcraft.jsch.Session? = null
    private var deployStartTime = 0L
    private var errorsFile: File? = null
    private val dateFormat = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())

    
    fun init(context: Context) {
        val dir = context.getExternalFilesDir(null) ?: context.filesDir
        errorsFile = File(dir, "errors.log")
    }

    fun getErrorsFile(): File? = errorsFile

    
    @Synchronized
    fun writeError(msg: String) {
        val file = errorsFile ?: return
        try {
            val timestamp = dateFormat.format(Date())
            file.appendText("[$timestamp] $msg\n")
            
            if (file.length() > 500_000) {
                val text = file.readText()
                file.writeText(text.takeLast(200_000))
            }
        } catch (_: Exception) { }
    }

    fun startDeploy() {
        
        if (isDeploying.value && deployStartTime > 0 &&
            System.currentTimeMillis() - deployStartTime > 30 * 60 * 1000) {
            writeError("Автосброс: предыдущий деплой завис >30 мин")
            forceReset()
        }
        isDeploying.value = true
        deployStartTime = System.currentTimeMillis()
        deployProgress.value = 0f
        currentStep.value = "Инициализация..."
        lastResult.value = ""
    }

    fun stopDeploy(result: String = "") {
        isDeploying.value = false
        deployStartTime = 0L
        if (result.isNotBlank()) lastResult.value = result
        val session = activeSession
        activeSession = null
        try { session?.disconnect() } catch (_: Exception) {}
    }

    
    fun forceReset() {
        val session = activeSession
        activeSession = null
        try { session?.disconnect() } catch (_: Exception) {}
        isDeploying.value = false
        deployStartTime = 0L
        deployProgress.value = 0f
        currentStep.value = ""
    }

    fun updateProgress(progress: Float, step: String) {
        deployProgress.value = progress
        currentStep.value = step
    }

    fun checkUpdate() {
        scope.launch {
            try {
                val url = URL("https://api.github.com/repos/$GITHUB_REPO/releases/latest")
                val conn = url.openConnection() as HttpURLConnection
                conn.setRequestProperty("Accept", "application/json")
                conn.connectTimeout = 10000
                conn.readTimeout = 10000
                val body = conn.inputStream.bufferedReader().readText()
                conn.disconnect()

                val tag = Regex("\"tag_name\"\\s*:\\s*\"([^\"]+)\"").find(body)?.groupValues?.get(1) ?: return@launch
                val current = currentVersion.value
                if (current.isEmpty() || tag != current) {
                    availableVersion.value = tag
                }
            } catch (_: Exception) {}
        }
    }

    suspend fun downloadUpdate(context: Context, onProgress: (Float, String) -> Unit): File? = withContext(Dispatchers.IO) {
        try {
            val version = availableVersion.value
            if (version.isEmpty()) return@withContext null

            onProgress(0.1f, "Скачивание $version...")
            val url = URL("https://github.com/$GITHUB_REPO/releases/download/$version/wdtt-server")
            val conn = url.openConnection() as HttpURLConnection
            conn.connectTimeout = 15000
            conn.readTimeout = 60000
            conn.connect()

            val total = conn.contentLengthLong
            val input = conn.inputStream
            val file = File(context.cacheDir, "wdtt-server-update")
            FileOutputStream(file).use { output ->
                val buf = ByteArray(8192)
                var downloaded = 0L
                var progress = 0
                var bytesRead: Int
                while (input.read(buf).also { bytesRead = it } != -1) {
                    output.write(buf, 0, bytesRead)
                    downloaded += bytesRead
                    if (total > 0) {
                        val p = downloaded.toFloat() / total
                        onProgress(0.1f + p * 0.4f, "Скачивание ${downloaded / 1024}KB / ${total / 1024}KB")
                        progress = (p * 40).toInt()
                    }
                }
            }
            conn.disconnect()
            onProgress(0.5f, "Скачано: ${file.length() / 1024}KB")
            file
        } catch (e: Exception) {
            writeError("Update download: ${e.message}")
            null
        }
    }

    fun startUpdate() {
        isUpdating.value = true
        deployProgress.value = 0f
        currentStep.value = "Обновление..."
    }

    fun stopUpdate(result: String = "") {
        isUpdating.value = false
        if (result.isNotBlank()) lastResult.value = result
    }
}
