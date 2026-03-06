# Plotix Mobile - Руководство по интеграции Go-ядра

## Общее описание

Plotix Core (Go) компилируется в нативную библиотеку через `gomobile`:
- **Android**: `.aar` файл
- **iOS**: `.xcframework`

Библиотека запускает полноценную P2P ноду (шифрование, обнаружение пиров, обмен сообщениями, WebRTC сигналинг) **прямо на устройстве**. Мобильный UI общается с ней через HTTP-запросы на `localhost:8080`.

---

## 1. Установка инструментов (пошагово)

Перед сборкой нужно установить 4 вещи. Ниже все ссылки и команды.

### 1.1 Go (язык программирования)

Скачать: https://go.dev/dl/

- Выбери **go1.23.x.windows-amd64.msi** (или новее, но 1.23 самая стабильная для gomobile)
- Установи, все галочки по умолчанию
- После установки открой **новый** терминал и проверь:
```powershell
go version
# Должно показать: go version go1.23.x windows/amd64
```

### 1.2 Python (для скрипта сборки)

Скачать: https://www.python.org/downloads/

- При установке **обязательно поставь галочку "Add Python to PATH"**
- Проверь:
```powershell
python --version
```

### 1.3 Android SDK + NDK

Самый простой способ — через Android Studio:

1. Скачать Android Studio: https://developer.android.com/studio
2. Установить, запустить
3. Перейти: **Settings (Ctrl+Alt+S) -> Languages & Frameworks -> Android SDK**
4. Вкладка **SDK Platforms**: убедись, что установлен хотя бы один Android API (например, API 34)
5. Вкладка **SDK Tools**: поставь галочки и установи:
   - **NDK (Side by side)** — ОБЯЗАТЕЛЬНО, без этого ничего не соберется
   - **CMake**
   - **Android SDK Build-Tools**
   - **Android SDK Command-line Tools**

После установки запомни путь к SDK (обычно `C:\Users\<ИМЯ>\AppData\Local\Android\Sdk`).

**Настройка переменных среды (Windows):**

1. Нажми Win, введи **"Изменение системных переменных среды"**, открой
2. Нажми **"Переменные среды..."**
3. В разделе **"Переменные пользователя"** нажми **"Создать"** и добавь:

| Имя переменной | Значение |
|----------------|----------|
| `ANDROID_HOME` | `C:\Users\<ИМЯ>\AppData\Local\Android\Sdk` |
| `ANDROID_NDK_HOME` | `C:\Users\<ИМЯ>\AppData\Local\Android\Sdk\ndk\<ВЕРСИЯ>` |

> Чтобы узнать версию NDK, зайди в папку `...\Android\Sdk\ndk\` — там будет папка типа `26.1.10909125`

4. Там же найди переменную **Path**, нажми **"Изменить"** и добавь строку:
```
%USERPROFILE%\go\bin
```
5. Нажми OK везде и **перезапусти VS Code / терминал**

### 1.4 gomobile (инструмент сборки Go -> мобилка)

Открой **новый** терминал (после настройки PATH!) и выполни:
```powershell
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
gomobile init
```

**Проверь, что gomobile доступен:**
```powershell
where gomobile
# Должно показать: C:\Users\<ИМЯ>\go\bin\gomobile.exe
```

### 1.5 Проверочный чеклист

Перед сборкой убедись, что все 4 команды работают:
```powershell
go version          # go1.23.x или новее
python --version    # Python 3.x
gomobile version    # не должна быть ошибка "not recognized"
echo %ANDROID_HOME% # путь к Android SDK (не пустой)
```

---

## 2. Сборка библиотек

Из корня проекта (`nuclear_hack/`):
```bash
# Только Android (на Windows iOS не собирается)
python build_mobile.py android

# Обе платформы (iOS только на macOS)
python build_mobile.py

# Только iOS (нужен macOS + Xcode)
python build_mobile.py ios
```

Результат будет в папке `mobile_out/`:
- `plotix_core.aar` (Android)
- `PlotixCore.xcframework` (iOS, только на macOS)

**Если сборка падает с ошибкой:**

| Ошибка | Решение |
|--------|---------|
| `gomobile: not recognized` | Добавь `%USERPROFILE%\go\bin` в PATH, перезапусти терминал |
| `could not locate Android SDK` | Проверь переменную `ANDROID_HOME`, она должна указывать на папку SDK |
| `no Android NDK found` | Установи NDK через Android Studio (SDK Tools), задай `ANDROID_NDK_HOME` |
| `go.mod: go 1.25.0` | В файле `plotix_core/go.mod` поменяй версию на `go 1.23` |

---

## 3. Интеграция в Android

> Для iOS-разработчика: сборка `.xcframework` возможна только на macOS. Нужны Xcode (https://apps.apple.com/app/xcode/id497799835) и те же Go + gomobile. Android NDK не нужен.

---

### 3.1 Добавление библиотеки
1. Скопируй `plotix_core.aar` в `composeApp/libs/`
2. Добавь в `composeApp/build.gradle.kts`:
```kotlin
dependencies {
    implementation(files("libs/plotix_core.aar"))
}
```

### 3.2 Запуск ядра
В `MainActivity.kt`:
```kotlin
import mobile.Mobile

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Запускаем Go-ядро, передаем путь к внутреннему хранилищу
        val result = Mobile.startNode(applicationContext.filesDir.absolutePath)
        if (result != "started") {
            Log.e("Plotix", "Ошибка запуска ядра: $result")
        }

        setContent { App() }
    }
}
```

**Что происходит после вызова `startNode()`:**
- Создается (или загружается) аккаунт с ключами шифрования
- Запускается HTTP API сервер на порту 8080
- Запускается P2P TCP сервер на порту 10000
- Запускается multicast-обнаружение пиров в локальной сети
- Автоматически выполняется handshake с найденными пирами

### 3.3 Необходимые разрешения
Добавь в `AndroidManifest.xml`:
```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
<uses-permission android:name="android.permission.ACCESS_WIFI_STATE" />
<uses-permission android:name="android.permission.CHANGE_WIFI_MULTICAST_STATE" />

<!-- Для видеозвонков (WebRTC) -->
<uses-permission android:name="android.permission.CAMERA" />
<uses-permission android:name="android.permission.RECORD_AUDIO" />
```

---

## 4. Интеграция в iOS

### 4.1 Добавление фреймворка
1. Перетащи `PlotixCore.xcframework` в Xcode проект
2. Убедись, что он добавлен в "Frameworks, Libraries, and Embedded Content" как "Embed & Sign"

### 4.2 Запуск ядра
В `iOSApp.swift` или `ContentView.swift`:
```swift
import Mobile

@main
struct iOSApp: App {
    init() {
        let dataDir = FileManager.default.urls(
            for: .documentDirectory,
            in: .userDomainMask
        )[0].path

        let result = MobileStartNode(dataDir)
        if result != "started" {
            print("Ошибка запуска ядра: \(result ?? "nil")")
        }
    }
    // ...
}
```

### 4.3 Info.plist
Добавь ключ `NSLocalNetworkUsageDescription` для доступа к локальной сети (multicast discovery).

---

## 5. API (localhost:8080)

После запуска ядра весь UI общается с ним через HTTP и WebSocket на `http://localhost:8080`.

### REST-эндпоинты

| Метод | Эндпоинт | Описание | Тело запроса |
|-------|----------|----------|--------------|
| GET | `/me` | Информация о текущем узле (PeerID, имя) | - |
| GET | `/peers` | Список обнаруженных пиров | - |
| GET | `/history?peer_id=XXX` | История чата с пиром | - |
| POST | `/send_message` | Отправить сообщение | `{"peer_id": "...", "text": "..."}` |
| POST | `/send_file` | Отправить файл | multipart/form-data: `file`, `peer_id` |
| GET | `/accounts` | Список всех аккаунтов | - |
| POST | `/accounts/create` | Создать новый аккаунт | `{"name": "..."}` |
| POST | `/accounts/switch` | Переключить аккаунт | `{"peer_id": "..."}` |
| POST | `/accounts/rename` | Переименовать аккаунт | `{"peer_id": "...", "name": "..."}` |
| POST | `/accounts/ghost` | Режим призрака (скрыть имя) | `{"peer_id": "...", "ghost": true}` |
| POST | `/peer/rename` | Задать алиас пиру | `{"peer_id": "...", "name": "..."}` |
| POST | `/add_peer` | Ручное подключение к пиру | `{"ip": "192.168.1.5"}` |
| POST | `/webrtc/signal` | WebRTC сигнал | `{"peer_id": "...", "type": "...", "data": "..."}` |
| GET | `/view?id=XXX` | Просмотр полученного файла | - |

### WebSocket-события

Подключение: `ws://localhost:8080/events`

Формат: JSON `{"type": "...", "payload": {...}}`

| Тип | Payload | Описание |
|-----|---------|----------|
| `new_message` | `{from, text, timestamp, id, ...}` | Входящее сообщение |
| `file_received` | `{from, filename, file_id, size}` | Входящий файл |
| `peer_online` | `{peer_id, name, ip}` | Пир подключился |
| `peer_offline` | `{peer_id}` | Пир отключился |
| `webrtc_signal` | `{from, type, data}` | Данные WebRTC сигналинга |
| `account_switched` | `{peer_id}` | Аккаунт переключен |

---

## 6. Примеры кода (Kotlin)

### Получение списка пиров
```kotlin
val client = OkHttpClient()
val request = Request.Builder()
    .url("http://localhost:8080/peers")
    .build()

client.newCall(request).enqueue(object : Callback {
    override fun onResponse(call: Call, response: Response) {
        val json = response.body?.string()
        // Парсим список пиров
    }
    override fun onFailure(call: Call, e: IOException) {
        Log.e("Plotix", "Ошибка получения пиров", e)
    }
})
```

### Отправка сообщения
```kotlin
val json = """{"peer_id": "$peerId", "text": "$messageText"}"""
val body = json.toRequestBody("application/json".toMediaType())
val request = Request.Builder()
    .url("http://localhost:8080/send_message")
    .post(body)
    .build()

client.newCall(request).enqueue(object : Callback {
    override fun onResponse(call: Call, response: Response) {
        // Сообщение отправлено
    }
    override fun onFailure(call: Call, e: IOException) {
        Log.e("Plotix", "Ошибка отправки", e)
    }
})
```

### Подписка на WebSocket-события
```kotlin
val wsClient = OkHttpClient()
val wsRequest = Request.Builder()
    .url("ws://localhost:8080/events")
    .build()

wsClient.newWebSocket(wsRequest, object : WebSocketListener() {
    override fun onMessage(webSocket: WebSocket, text: String) {
        // Парсим JSON: {"type": "new_message", "payload": {...}}
        val event = Json.decodeFromString<WSEvent>(text)
        when (event.type) {
            "new_message" -> { /* обновить UI чата */ }
            "peer_online" -> { /* добавить пира в список */ }
            "peer_offline" -> { /* пометить пира как оффлайн */ }
            "file_received" -> { /* показать уведомление о файле */ }
        }
    }

    override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
        Log.e("Plotix", "WebSocket ошибка", t)
        // Переподключение через N секунд
    }
})
```

### Получение истории чата
```kotlin
val request = Request.Builder()
    .url("http://localhost:8080/history?peer_id=$peerId")
    .build()

client.newCall(request).enqueue(object : Callback {
    override fun onResponse(call: Call, response: Response) {
        val json = response.body?.string()
        // Парсим массив сообщений
    }
    override fun onFailure(call: Call, e: IOException) {
        Log.e("Plotix", "Ошибка загрузки истории", e)
    }
})
```

---

## 7. Архитектура

```
+---------------------------+
|   Compose Multiplatform   |
|   (Kotlin UI)             |
|                           |
|  HTTP -> localhost:8080   |
|  WS   -> localhost:8080   |
+---------------------------+
            |
            v
+---------------------------+
|   Go Core (in-process)    |
|                           |
|  - API Server (порт 8080) |
|  - P2P TCP (порт 10000)   |
|  - Multicast Discovery    |
|  - E2EE (X25519/AES-GCM)  |
|  - BoltDB Storage         |
+---------------------------+
            |
            v
+---------------------------+
|   Другие ноды Plotix      |
|   (LAN / прямой TCP)      |
+---------------------------+
```

---

## 8. Важные замечания

- **Нет внешних серверов**: все работает на устройстве. Обнаружение пиров через multicast/broadcast в локальной сети.
- **E2EE**: все сообщения шифруются X25519 (обмен ключами) + AES-256-GCM. Ключи хранятся в `dataDir/<peerID>/keystore.json`.
- **Порт 8080**: API слушает на `0.0.0.0:8080`. На Android доступен только с самого устройства (localhost).
- **Порт 10000**: P2P TCP порт. Убедись, что приложение имеет доступ к локальной сети.
- **Фоновый сервис**: для стабильной работы P2P рекомендуется обернуть `Mobile.startNode()` в Foreground Service на Android, чтобы система не убивала процесс.
- **Ограничения gomobile**: через границу Go-мобилка могут проходить только примитивные типы (`string`, `int`, `bool`, `[]byte`). Вся сложная логика идет через HTTP API.
- **Дополнительные функции Go-моста**:
  - `Mobile.getPeerID()` — получить PeerID текущего узла
  - `Mobile.getAPIPort()` — получить порт API ("8080")
