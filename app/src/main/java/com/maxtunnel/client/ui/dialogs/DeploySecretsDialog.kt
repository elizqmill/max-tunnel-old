package com.maxtunnel.client.ui.dialogs

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.maxtunnel.client.SettingsStore
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeploySecretsDialog(
    settingsStore: SettingsStore,
    initialMainPass: String,
    initialAdminId: String,
    initialBotToken: String,
    initialSshPort: String,
    manualPortsEnabled: Boolean,
    initialServerDtlsPort: String,
    initialServerWgPort: String,
    onSaved: (String, String) -> Unit,
    onDismiss: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var passInput by rememberSaveable { mutableStateOf(initialMainPass) }
    var adminIdInput by rememberSaveable { mutableStateOf(initialAdminId) }
    var botTokenInput by rememberSaveable { mutableStateOf(initialBotToken) }
    var sshPortInput by rememberSaveable { mutableStateOf(if (initialSshPort.isBlank()) "22" else initialSshPort) }
    var dtlsPortInput by rememberSaveable { mutableStateOf(initialServerDtlsPort.ifBlank { "56000" }) }
    var wgPortInput by rememberSaveable { mutableStateOf(initialServerWgPort.ifBlank { "56001" }) }

    fun normalizePort(value: String, fallback: String): String {
        return value.toIntOrNull()?.takeIf { it in 1..65535 }?.toString() ?: fallback
    }

    Dialog(onDismissRequest = onDismiss) {
        Surface(
            shape = RoundedCornerShape(24.dp),
            color = MaterialTheme.colorScheme.surface,
            contentColor = MaterialTheme.colorScheme.onSurface,
            tonalElevation = 8.dp
        ) {
            Column(modifier = Modifier.padding(24.dp).fillMaxWidth().verticalScroll(rememberScrollState())) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("Секреты Деплоя", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                    IconButton(onClick = onDismiss) {
                        Icon(Icons.Default.Close, contentDescription = "Закрыть")
                    }
                }

                Spacer(Modifier.height(16.dp))

                val isPasswordValid = passInput.isNotEmpty() && passInput.matches(Regex("^[a-zA-Z0-9_.!?:#/-]+$"))

                OutlinedTextField(
                    value = passInput,
                    onValueChange = { passInput = it.filter { c -> !c.isWhitespace() } },
                    label = { Text("Задайте пароль туннеля (любой)") },
                    placeholder = { Text("Придумайте надежный пароль") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    isError = passInput.isNotEmpty() && !isPasswordValid,
                    supportingText = if (passInput.isNotEmpty() && !isPasswordValid) {
                        { Text("Разрешены только буквы, цифры и симв: _ . ! ? : # - /", color = MaterialTheme.colorScheme.error) }
                    } else null
                )

                Spacer(Modifier.height(16.dp))
                HorizontalDivider()
                Spacer(Modifier.height(8.dp))
                Text("Телеграм бот для управления", color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
                Spacer(Modifier.height(8.dp))

                OutlinedTextField(
                    value = adminIdInput,
                    onValueChange = { adminIdInput = it },
                    label = { Text("ID Админа (Опционально)") },
                    placeholder = { Text("ID из @getmyid_bot") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Number
                    )
                )

                Spacer(Modifier.height(8.dp))

                OutlinedTextField(
                    value = botTokenInput,
                    onValueChange = { botTokenInput = it },
                    label = { Text("Токен Бота (Опционально)") },
                    placeholder = { Text("Токен от BotFather") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp)
                )

                Spacer(Modifier.height(16.dp))
                HorizontalDivider()
                Spacer(Modifier.height(8.dp))
                Text("SSH Порт", color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
                Spacer(Modifier.height(8.dp))

                OutlinedTextField(
                    value = sshPortInput,
                    onValueChange = { sshPortInput = it.filter(Char::isDigit).take(5) },
                    label = { Text("Порт для деплоя SSH") },
                    placeholder = { Text("22") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Number
                    )
                )

                if (manualPortsEnabled) {
                    Spacer(Modifier.height(16.dp))
                    HorizontalDivider()
                    Spacer(Modifier.height(8.dp))
                    Text("Порты сервера", color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
                    Spacer(Modifier.height(8.dp))
                    OutlinedTextField(
                        value = dtlsPortInput,
                        onValueChange = { dtlsPortInput = it.filter(Char::isDigit).take(5) },
                        label = { Text("Порт DTLS сервера") },
                        placeholder = { Text("56000") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Number
                        )
                    )
                    Spacer(Modifier.height(8.dp))
                    OutlinedTextField(
                        value = wgPortInput,
                        onValueChange = { wgPortInput = it.filter(Char::isDigit).take(5) },
                        label = { Text("Порт WireGuard сервера") },
                        placeholder = { Text("56001") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Number
                        )
                    )
                }

                Spacer(Modifier.height(24.dp))
                Button(
                    onClick = {
                        val finalPort = if (sshPortInput.isBlank()) "22" else sshPortInput
                        val finalDtls = normalizePort(dtlsPortInput, "56000")
                        val finalWg = normalizePort(wgPortInput, "56001")
                        scope.launch {
                            settingsStore.saveDeploySecrets(passInput, adminIdInput, botTokenInput, finalPort)
                            settingsStore.savePorts(finalDtls.toInt(), finalWg.toInt(), settingsStore.listenPort.first())
                            onSaved(finalDtls, finalWg)
                            onDismiss()
                        }
                    },
                    modifier = Modifier.fillMaxWidth().height(48.dp),
                    shape = RoundedCornerShape(16.dp),
                    enabled = isPasswordValid,
                    colors = ButtonDefaults.buttonColors(contentColor = MaterialTheme.colorScheme.onPrimary)
                ) { Text("Сохранить", fontWeight = FontWeight.SemiBold) }
            }
        }
    }
}
