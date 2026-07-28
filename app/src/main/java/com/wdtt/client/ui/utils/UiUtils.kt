package com.wdtt.client.ui.utils

fun stripVkUrlStatic(input: String): String {
    var s = input.trim()
    val lower = s.lowercase()
    val prefixes = listOf(
        "https://max.ru/joincall/",
        "http://max.ru/joincall/",
        "max.ru/joincall/",
        "https://vk.com/call/join/",
        "http://vk.com/call/join/",
        "https://m.vk.com/call/join/",
        "http://m.vk.com/call/join/",
        "m.vk.com/call/join/",
        "vk.com/call/join/"
    )
    for (prefix in prefixes) {
        if (lower.startsWith(prefix)) {
            s = s.substring(prefix.length)
            break
        }
    }
    val qIdx = s.indexOf('?')
    if (qIdx != -1) s = s.substring(0, qIdx)
    val hIdx = s.indexOf('#')
    if (hIdx != -1) s = s.substring(0, hIdx)
    return s.trimEnd('/')
}

fun isMaxCallLink(input: String): Boolean {
    val lower = input.trim().lowercase()
    return lower.contains("max.ru/joincall/") || lower.contains("joincall/")
}

fun extractMaxCallLink(input: String): String {
    return stripVkUrlStatic(input)
}
