# MaxTunnel — WireGuard over DTLS + TURN

Android-приложение для создания VPN-туннеля поверх DTLS через TURN relay
с обфускацией трафика под RTP (audio/video/datachannel).

## Как это работает

1. Приложение авторизуется через VK или Max API, получает TURN-credentials
2. Поднимает DTLS-соединение к вашему VPS через TURN relay
3. Трафик обфусцируется под RTP-пакеты аудио/видео (или raw DTLS)
4. Сервер на VPS проксирует DTLS в userspace WireGuard
5. На устройстве поднимается локальный WireGuard VPN (VpnService + GoBackend)

## Возможности

- **Обфускация**: Audio (PT=111), Video (PT=96), Data Channel (raw DTLS)
- **Капча**: автоматический решатор VK Captcha (RJS/WBV)
- **Профили**: сохранение нескольких конфигураций подключения
- **Деплой**: установка сервера на VPS через SSH прямо из приложения
- **Обновление сервера**: авто-обновление бинарника через GitHub Releases

## Сборка

```bash
# Go-бинарник для Android
./scripts/build-go-lib.sh arm64-v8a

# APK (через Gradle)
./gradlew assembleRelease
```

## Репозитории

- **Сервер**: [elizqmill/maxtunnel-server](https://github.com/elizqmill/maxtunnel-server)
- **Клиент (Max)**: [elizqmill/Komet](https://github.com/elizqmill/Komet)

## Лицензия

Исходный код: [amurcanov/proxy-turn-vk-android](https://github.com/amurcanov/proxy-turn-vk-android)
