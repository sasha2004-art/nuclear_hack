<script setup>
import { ref, computed, nextTick, watch, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import { useChatStore } from '../stores/chat';

const route = useRoute();
const store = useChatStore();
const peerId = computed(() => decodeURIComponent(route.params.id));
const messages = computed(() => store.messages[peerId.value] || []);
const peerInfo = computed(() => store.peers[peerId.value] || {});
const peerDisplayName = computed(() => peerInfo.value.name || peerId.value);

const messageText = ref('');
const scrollContainer = ref(null);
const isRenaming = ref(false);
const newPeerName = ref('');
const fileInput = ref(null);

// Состояние для Drag & Drop - используем счетчик, чтобы избежать мерцания
const dragCounter = ref(0);
const isDragging = computed(() => dragCounter.value > 0);

// API URL для доступа к файлам
const API_URL = 'http://localhost:8080';

// Кэш для содержимого текстовых файлов
const textPreviews = ref({});

// WebRTC переменные
const localVideo = ref(null);
const remoteVideo = ref(null);
const isCalling = ref(false);
const isIncomingCall = ref(false);
const callType = ref('video'); // 'video' или 'audio'
const incomingSignal = ref(null); // Сохраненный оффер для принятия
const remoteStreamActive = ref(false); // Флаг активного видеопотока
const peerConnection = ref(null);

// Конфигурация WebRTC (используем только публичные Google STUN для определения внешних IP)
const rtcConfig = {
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
};

// Получить расширение файла
const getFileExt = (path) => path.split('.').pop().toLowerCase();

// Группировка расширений по типам
const types = {
    image: ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'],
    video: ['mp4', 'webm', 'ogg', 'mov'],
    audio: ['mp3', 'wav', 'ogg', 'm4a'],
    pdf: ['pdf'],
    text: ['txt', 'md', 'csv', 'json', 'log', 'js', 'py', 'go', 'html', 'css', 'sql', 'yaml', 'xml'],
    office: ['doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'odt', 'ods'],
    archive: ['zip', 'rar', '7z', 'tar', 'gz']
};

// Проверка типа файла
const isType = (path, category) => types[category] && types[category].includes(getFileExt(path));

// Загрузка текста для превью
const loadTextPreview = async (path) => {
    if (textPreviews.value[path]) return;
    try {
        const res = await fetch(getFileUrl(path));
        const text = await res.text();
        // Ограничиваем превью первыми 2000 символами
        textPreviews.value[path] = text.slice(0, 2000);
    } catch (e) {
        textPreviews.value[path] = 'Ошибка загрузки содержимого...';
    }
};

// Ссылка для отображения файла
const getFileUrl = (path) => `${API_URL}/view?path=${encodeURIComponent(path)}`;

// Парсим сообщение. Возвращает объект с путем, если это уведомление о файле
const parseFileMessage = (text) => {
    let path = '';
    let type = '';

    if (text.startsWith('[ФАЙЛ ПОЛУЧЕН] ')) {
        path = text.replace('[ФАЙЛ ПОЛУЧЕН] ', '').trim();
        type = 'received';
    } else if (text.startsWith('[ФАЙЛ ОТПРАВЛЕН] ')) {
        path = text.replace('[ФАЙЛ ОТПРАВЛЕН] ', '').trim();
        type = 'sent';
    } else {
        return null;
    }

    if (isType(path, 'text')) loadTextPreview(path);
    return { type, path };
};

const scrollToBottom = async () => {
    await nextTick();
    if (scrollContainer.value) {
        scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight;
    }
};

watch(messages, () => scrollToBottom(), { deep: true });
watch(peerId, (newId) => {
    if (newId) {
        store.fetchHistory(newId);
    }
    scrollToBottom();
}, { immediate: true });

const startRenaming = () => {
    newPeerName.value = peerInfo.value.name || '';
    isRenaming.value = true;
    nextTick(() => {
        document.getElementById('renameInput')?.focus();
    });
};

const saveName = async () => {
    await store.renamePeerLocal(peerId.value, newPeerName.value);
    isRenaming.value = false;
};

const handleSend = async () => {
    if (!messageText.value.trim()) return;
    const success = await store.sendMessage(peerId.value, messageText.value);
    if (success) {
        messageText.value = '';
    }
};

// --- Логика файлов ---
const uploadFile = async (file) => {
    if (!file) return;
    const success = await store.sendFile(peerId.value, file);
    if (!success) alert("Ошибка при добавлении файла в очередь");
    if (fileInput.value) fileInput.value.value = ''; // Сброс
};

const handleFileUpload = (event) => uploadFile(event.target.files[0]);

const onDragEnter = (e) => {
    e.preventDefault();
    dragCounter.value++;
};

const onDragLeave = (e) => {
    e.preventDefault();
    dragCounter.value--;
};

const onDrop = (e) => {
    e.preventDefault();
    dragCounter.value = 0; // Сбрасываем счетчик при дропе
    if (e.dataTransfer.files.length > 0) {
        uploadFile(e.dataTransfer.files[0]);
    }
};

// --- WebRTC логика ---
const setupWebRTC = async (type = 'video') => {
    callType.value = type;
    peerConnection.value = new RTCPeerConnection(rtcConfig);

    // Обработка найденных ICE-кандидатов
    peerConnection.value.onicecandidate = (event) => {
        if (event.candidate) {
            store.sendWebRTCSignal(peerId.value, 'candidate', event.candidate);
        }
    };

    // При получении удаленного потока
    peerConnection.value.ontrack = (event) => {
        if (remoteVideo.value) {
            remoteVideo.value.srcObject = event.streams[0];
        }
    };

    // Настройка ограничений (видео или только аудио)
    const constraints = {
        video: type === 'video' ? { width: 1280, height: 720 } : false,
        audio: true
    };

    try {
        const stream = await navigator.mediaDevices.getUserMedia(constraints);
        if (localVideo.value) localVideo.value.srcObject = stream;
        stream.getTracks().forEach(track => peerConnection.value.addTrack(track, stream));
    } catch (e) {
        console.error("Камера/микрофон недоступны", e);
    }
};

const startCall = async (type = 'video') => {
    // ЗАЩИТА: Если уже звоним или принимаем — выходим
    if (isCalling.value || isIncomingCall.value) return;

    isCalling.value = true;
    await setupWebRTC(type);
    const offer = await peerConnection.value.createOffer();
    await peerConnection.value.setLocalDescription(offer);
    store.sendWebRTCSignal(peerId.value, 'offer', { sdp: offer, callType: type });
};

const acceptCall = async () => {
    const data = JSON.parse(incomingSignal.value.data);
    isIncomingCall.value = false;
    isCalling.value = true;

    await setupWebRTC(data.callType || 'video');
    await peerConnection.value.setRemoteDescription(new RTCSessionDescription(data.sdp));

    const answer = await peerConnection.value.createAnswer();
    await peerConnection.value.setLocalDescription(answer);
    store.sendWebRTCSignal(peerId.value, 'answer', answer);
};

const rejectCall = () => {
    store.sendWebRTCSignal(peerId.value, 'reject', {});
    isIncomingCall.value = false;
    incomingSignal.value = null;
};

const cleanupCall = () => {
    if (peerConnection.value) {
        peerConnection.value.close();
        peerConnection.value = null;
    }
    if (localVideo.value?.srcObject) {
        localVideo.value.srcObject.getTracks().forEach(t => t.stop());
    }
    isCalling.value = false;
    isIncomingCall.value = false;
    remoteStreamActive.value = false;
};

const handleVideoPlaying = () => {
    if (callType.value === 'video') {
        remoteStreamActive.value = true;
    }
};

const endCall = () => {
    store.sendWebRTCSignal(peerId.value, 'hangup', {});
    cleanupCall();
};

// Слушатель сигналов от ядра
const handleSignal = async (e) => {
    const signal = e.detail;
    if (signal.sender_id !== peerId.value) return;

    if (signal.type === 'offer') {
        incomingSignal.value = signal;
        const data = JSON.parse(signal.data);
        callType.value = data.callType || 'video';
        isIncomingCall.value = true;
    } else if (signal.type === 'answer') {
        const data = JSON.parse(signal.data);
        await peerConnection.value.setRemoteDescription(new RTCSessionDescription(data));
    } else if (signal.type === 'candidate') {
        const data = JSON.parse(signal.data);
        await peerConnection.value.addIceCandidate(new RTCIceCandidate(data));
    } else if (signal.type === 'reject' || signal.type === 'hangup') {
        cleanupCall();
    }
};

window.addEventListener('webrtc-signal', handleSignal);
onUnmounted(() => window.removeEventListener('webrtc-signal', handleSignal));
</script>

<template>
    <!-- Главный контейнер оборачиваем в обработчики Drag & Drop -->
    <div
        class="flex flex-col h-full relative"
        @dragenter.prevent="onDragEnter"
        @dragleave.prevent="onDragLeave"
        @dragover.prevent
        @drop.prevent="onDrop"
    >
        <!-- Overlay при перетаскивании -->
        <!-- pointer-events-none КРИТИЧЕСКИ ВАЖЕН: он заставляет оверлей "пропускать" события сквозь себя -->
        <div
            v-if="isDragging"
            class="pointer-events-none absolute inset-0 bg-indigo-600/40 border-4 border-dashed border-indigo-400 z-50 flex flex-col items-center justify-center transition-all duration-200 backdrop-blur-md"
        >
            <div class="bg-gray-900 p-8 rounded-3xl shadow-2xl flex flex-col items-center gap-4 transform scale-110">
                <div class="w-20 h-20 bg-indigo-500 rounded-full flex items-center justify-center animate-bounce">
                    <svg class="w-10 h-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                    </svg>
                </div>
                <span class="text-xl font-bold text-white tracking-tight">Отпустите для отправки</span>
                <span class="text-sm text-gray-400">Файлы до 1 ГБ • E2EE защита</span>
            </div>
        </div>

        <header class="p-4 bg-gray-800 shadow-md flex justify-between items-center z-10">
            <div class="flex-1">
                <div v-if="!isRenaming" class="flex items-center gap-2 group">
                    <h2 class="font-bold text-gray-200 text-lg">{{ peerDisplayName }}</h2>
                    <button @click="startRenaming" class="text-gray-500 hover:text-indigo-400 opacity-0 group-hover:opacity-100 transition p-1">
                        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                        </svg>
                    </button>
                </div>

                <div v-else class="flex items-center gap-2">
                    <input
                        id="renameInput"
                        v-model="newPeerName"
                        @keyup.enter="saveName"
                        @blur="isRenaming = false"
                        class="bg-gray-700 text-white px-2 py-1 rounded text-sm border border-indigo-500 outline-none"
                        placeholder="Имя контакта..."
                    />
                    <button @mousedown.prevent="saveName" class="text-green-400 text-sm">OK</button>
                </div>

                <p v-if="peerInfo.name && peerInfo.name !== peerId" class="text-xs text-gray-500 font-mono mt-1">
                    ID: {{ peerId.slice(0, 12) }}...
                </p>
                <p class="text-[10px] text-gray-600">DIRECT TCP + OFFLINE QUEUE</p>
            </div>

            <!-- WebRTC звонки кнопки (скрываются, если вызов уже в процессе) -->
            <div v-if="!isCalling && !isIncomingCall" class="flex gap-2 animate-in fade-in zoom-in duration-300">
                <button
                    @click="startCall('audio')"
                    class="p-3 bg-gray-700 hover:bg-indigo-600 rounded-full text-white transition shadow-lg active:scale-90"
                    title="Голосовой звонок"
                >
                    <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
                    </svg>
                </button>
                <button
                    @click="startCall('video')"
                    class="p-3 bg-gray-700 hover:bg-emerald-600 rounded-full text-white transition shadow-lg active:scale-90"
                    title="Видеозвонок"
                >
                    <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                </button>
            </div>

            <!-- Индикатор того, что линия занята -->
            <div v-else class="px-4 py-2 bg-indigo-500/10 border border-indigo-500/20 rounded-full flex items-center gap-2">
                <div class="w-2 h-2 rounded-full bg-indigo-500 animate-pulse"></div>
                <span class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest">Линия занята</span>
            </div>
        </header>

        <div ref="scrollContainer" class="flex-1 overflow-y-auto p-6 space-y-4 bg-gray-900 z-0">
            <div v-for="(msg, idx) in messages" :key="msg.id || idx"
                :class="['flex', msg.self ? 'justify-end' : 'justify-start']">

                <div :class="[
                    'max-w-[80%] p-1 rounded-2xl text-sm shadow-sm overflow-hidden',
                    msg.self ? 'bg-indigo-600 text-white rounded-tr-none' : 'bg-gray-700 text-gray-200 rounded-tl-none'
                ]">

                    <!-- ЛОГИКА ОТОБРАЖЕНИЯ КОНТЕНТА -->
                    <template v-if="parseFileMessage(msg.text)">
                        <div class="p-1 min-w-[240px]">
                            <div class="flex items-center gap-2 mb-2 px-2 pt-1">
                                <!-- Иконка статуса (Стрелочка вверх/вниз) -->
                                <svg v-if="parseFileMessage(msg.text).type === 'sent'" class="w-3 h-3 text-indigo-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
                                </svg>
                                <svg v-else class="w-3 h-3 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
                                </svg>

                                <span class="text-[10px] font-bold uppercase tracking-wider opacity-60">
                                    {{ getFileExt(parseFileMessage(msg.text).path) }}
                                </span>
                            </div>

                            <!-- ПРЕДПРОСМОТР КОНТЕНТА -->
                            <div class="rounded-xl overflow-hidden bg-black/20 border border-white/5">

                                <!-- Картинки -->
                                <img v-if="isType(parseFileMessage(msg.text).path, 'image')"
                                     :src="getFileUrl(parseFileMessage(msg.text).path)"
                                     class="max-w-full max-h-80 object-contain cursor-pointer"
                                     @click="window.open(getFileUrl(parseFileMessage(msg.text).path))" />

                                <!-- Текст / Код -->
                                <div v-else-if="isType(parseFileMessage(msg.text).path, 'text')" class="p-3 font-mono text-[11px] max-h-48 overflow-hidden relative">
                                    <pre class="text-indigo-100/80 whitespace-pre-wrap break-all">{{ textPreviews[parseFileMessage(msg.text).path] || 'Загрузка...' }}</pre>
                                    <div class="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-gray-800 to-transparent"></div>
                                </div>

                                <!-- PDF -->
                                <iframe v-else-if="isType(parseFileMessage(msg.text).path, 'pdf')" :src="getFileUrl(parseFileMessage(msg.text).path)" class="w-full h-64 border-none"></iframe>

                                <!-- Видео -->
                                <video v-else-if="isType(parseFileMessage(msg.text).path, 'video')" controls class="w-full max-h-64">
                                    <source :src="getFileUrl(parseFileMessage(msg.text).path)">
                                </video>

                                <!-- Аудио -->
                                <div v-else-if="isType(parseFileMessage(msg.text).path, 'audio')" class="p-4">
                                    <audio controls class="w-full h-8">
                                        <source :src="getFileUrl(parseFileMessage(msg.text).path)">
                                    </audio>
                                </div>

                                <!-- Общая карточка для Документов/Таблиц/Архивов -->
                                <div v-else class="flex items-center gap-4 p-4 bg-white/5">
                                    <!-- Иконка файла (Лист бумаги) -->
                                    <div class="w-10 h-10 rounded-lg bg-white/10 flex items-center justify-center shrink-0">
                                        <svg class="w-6 h-6 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                        </svg>
                                    </div>
                                    <div class="flex-1 min-w-0">
                                        <div class="truncate font-medium text-sm">{{ parseFileMessage(msg.text).path.split(/[\\/]/).pop() }}</div>
                                        <div class="text-[10px] opacity-40 uppercase tracking-tighter">{{ getFileExt(parseFileMessage(msg.text).path) }} Document</div>
                                    </div>
                                    <a :href="getFileUrl(parseFileMessage(msg.text).path)" download class="p-2 rounded-full bg-white/10 hover:bg-indigo-500 transition text-white">
                                        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                                        </svg>
                                    </a>
                                </div>
                            </div>
                        </div>
                    </template>

                    <!-- ОБЫЧНЫЙ ТЕКСТ -->
                    <template v-else>
                        <div class="px-3 py-2">
                            <p class="whitespace-pre-wrap">{{ msg.text }}</p>
                        </div>
                    </template>

                    <!-- ВРЕМЯ -->
                    <span class="text-[10px] opacity-40 px-3 pb-1 block text-right">
                        {{ new Date(msg.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}) }}
                    </span>
                </div>
            </div>
        </div>

        <footer class="p-4 bg-gray-800 border-t border-gray-700 z-10">
            <div class="max-w-4xl mx-auto flex gap-4 items-center">
                <!-- Кнопка прикрепления файла -->
                <button
                    @click="$refs.fileInput.click()"
                    class="bg-gray-800 hover:bg-gray-700 text-indigo-400 w-12 h-12 rounded-2xl flex items-center justify-center transition-all border border-gray-700 hover:border-indigo-500/50 shrink-0 cursor-pointer group"
                    title="Прикрепить файл"
                >
                    <svg class="w-6 h-6 group-hover:scale-110 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                    </svg>
                </button>
                <input
                    type="file"
                    ref="fileInput"
                    class="hidden"
                    @change="handleFileUpload"
                />

                <input
                    v-model="messageText"
                    @keyup.enter="handleSend"
                    type="text"
                    placeholder="Введите сообщение..."
                    class="flex-1 bg-gray-900 border border-gray-700 rounded-xl px-4 py-3 text-gray-100 focus:outline-none focus:border-indigo-500 transition"
                />
                <button
                    @click="handleSend"
                    class="bg-indigo-600 hover:bg-indigo-500 text-white px-6 py-3 rounded-xl font-bold transition shadow-lg active:transform active:scale-95 shrink-0"
                >
                    SEND
                </button>
            </div>
        </footer>

        <!-- UI: ВХОДЯЩИЙ ЗВОНОК (Glassmorphism Modal) -->
        <transition name="fade">
            <div v-if="isIncomingCall" class="absolute inset-0 z-[60] flex items-center justify-center bg-gray-950/80 backdrop-blur-md">
                <div class="bg-gray-800 p-8 rounded-[40px] border border-white/10 shadow-2xl flex flex-col items-center text-center max-w-sm w-full mx-4">
                    <div class="relative mb-6">
                        <div class="absolute inset-0 rounded-full bg-indigo-500 animate-ping opacity-20"></div>
                        <div class="w-24 h-24 rounded-full bg-indigo-600 flex items-center justify-center text-3xl font-bold text-white shadow-xl">
                            {{ peerDisplayName.charAt(0).toUpperCase() }}
                        </div>
                    </div>
                    <h3 class="text-2xl font-bold text-white mb-1">{{ peerDisplayName }}</h3>
                    <p class="text-gray-400 mb-8">{{ callType === 'video' ? 'Видеозвонок...' : 'Аудиозвонок...' }}</p>

                    <div class="flex gap-6 w-full">
                        <button @click="rejectCall" class="flex-1 py-4 bg-rose-500 hover:bg-rose-600 rounded-2xl text-white font-bold transition-all active:scale-95 shadow-lg shadow-rose-500/20">
                            Сбросить
                        </button>
                        <button @click="acceptCall" class="flex-1 py-4 bg-emerald-500 hover:bg-emerald-600 rounded-2xl text-white font-bold transition-all active:scale-95 shadow-lg shadow-emerald-500/20">
                            Принять
                        </button>
                    </div>
                </div>
            </div>
        </transition>

        <!-- UI: АКТИВНЫЙ ЗВОНОК (Аудио или Видео) -->
        <transition name="fade">
            <div v-if="isCalling" class="absolute inset-0 z-50 bg-gray-950 flex flex-col overflow-hidden">
                <div class="relative flex-1 bg-black flex items-center justify-center">

                    <!-- Видео собеседника -->
                    <video
                        v-show="callType === 'video'"
                        ref="remoteVideo"
                        autoplay
                        playsinline
                        @playing="handleVideoPlaying"
                        class="w-full h-full object-cover transition-opacity duration-700"
                        :class="remoteStreamActive ? 'opacity-100' : 'opacity-0'"
                    ></video>

                    <!-- Оверлей аватара (Показываем только если Аудио-звонок ИЛИ видео еще не подгрузилось) -->
                    <div v-if="callType === 'audio' || !remoteStreamActive"
                         class="absolute inset-0 flex flex-col items-center justify-center bg-gray-900/40 backdrop-blur-sm">

                        <div class="relative">
                            <!-- Красивая пульсация (только если видео еще нет или это аудио) -->
                            <div class="absolute inset-0 rounded-full bg-indigo-500/20 animate-ping" style="animation-duration: 3s"></div>
                            <div class="absolute inset-[-20px] rounded-full border border-indigo-500/10 animate-pulse"></div>

                            <!-- Аватар -->
                            <div class="w-40 h-40 rounded-full bg-gray-800 border-4 border-gray-700 shadow-2xl flex items-center justify-center relative z-10">
                                <span class="text-7xl font-bold text-white/90 drop-shadow-lg">
                                    {{ peerDisplayName.charAt(0).toUpperCase() }}
                                </span>
                            </div>
                        </div>

                        <div class="mt-10 text-center">
                            <h3 class="text-xl font-bold text-white mb-2">{{ peerDisplayName }}</h3>
                            <p class="text-xs text-indigo-400 font-medium tracking-[0.2em] uppercase opacity-70">
                                {{ callType === 'audio' ? 'Голосовая связь' : 'Установка видеосвязи...' }}
                            </p>
                        </div>
                    </div>

                    <!-- Твоё превью (PIP) -->
                    <div v-if="callType === 'video'"
                         class="absolute top-6 right-6 w-44 aspect-video rounded-2xl overflow-hidden border-2 border-white/10 shadow-2xl bg-gray-900 ring-1 ring-black/50">
                        <video ref="localVideo" autoplay muted playsinline class="w-full h-full object-cover"></video>
                    </div>
                </div>

                <!-- Кнопка сброса (Более аккуратная) -->
                <div class="absolute bottom-12 left-1/2 -translate-x-1/2">
                    <button @click="endCall"
                            class="group relative w-16 h-16 flex items-center justify-center bg-red-500 rounded-full shadow-[0_0_30px_rgba(239,68,68,0.4)] transition-all hover:bg-red-600 active:scale-90">
                        <svg class="w-8 h-8 text-white rotate-[135deg] group-hover:scale-110 transition-transform" fill="currentColor" viewBox="0 0 20 20">
                            <path d="M2 3a1 1 0 011-1h2.153a1 1 0 01.986.836l.74 4.435a1 1 0 01-.54 1.06l-1.548.773a11.037 11.037 0 006.105 6.105l.774-1.548a1 1 0 011.059-.54l4.435.74a1 1 0 01.836.986V17a1 1 0 01-1 1h-2C7.82 18 2 12.18 2 5V3z" />
                        </svg>
                    </button>
                </div>
            </div>
        </transition>
    </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active {
    transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.fade-enter-from, .fade-leave-to {
    opacity: 0;
    transform: scale(1.05);
}

/* Эффект свечения для видео */
video {
    mask-image: radial-gradient(white, black);
}
</style>
