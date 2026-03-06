<script setup>
import { ref, computed, nextTick, watch } from 'vue';
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
    </div>
</template>
