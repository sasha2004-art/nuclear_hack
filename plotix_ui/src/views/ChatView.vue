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
</script>

<template>
    <div class="flex flex-col h-full">

        <header class="p-4 bg-gray-800 shadow-md flex justify-between items-center">
            <div class="flex-1">
                <div v-if="!isRenaming" class="flex items-center gap-2 group">
                    <h2 class="font-bold text-gray-200 text-lg">{{ peerDisplayName }}</h2>
                    <button @click="startRenaming" class="text-gray-500 hover:text-indigo-400 opacity-0 group-hover:opacity-100 transition">
                        &#9998;
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
                <p class="text-[10px] text-gray-600">DIRECT TCP</p>
            </div>
        </header>

        <div ref="scrollContainer" class="flex-1 overflow-y-auto p-6 space-y-4 bg-gray-900">
            <div v-for="(msg, idx) in messages" :key="msg.id || idx"
                :class="['flex', msg.self ? 'justify-end' : 'justify-start']">
                <div :class="[
                    'max-w-[70%] p-3 rounded-2xl text-sm shadow-sm',
                    msg.self ? 'bg-indigo-600 text-white rounded-tr-none' : 'bg-gray-700 text-gray-200 rounded-tl-none'
                ]">
                    <p>{{ msg.text }}</p>
                    <span class="text-[10px] opacity-50 mt-1 block text-right">
                        {{ new Date(msg.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}) }}
                    </span>
                </div>
            </div>
        </div>

        <footer class="p-4 bg-gray-800 border-t border-gray-700">
            <div class="max-w-4xl mx-auto flex gap-4">
                <input
                    v-model="messageText"
                    @keyup.enter="handleSend"
                    type="text"
                    placeholder="Введите сообщение..."
                    class="flex-1 bg-gray-900 border border-gray-700 rounded-xl px-4 py-3 text-gray-100 focus:outline-none focus:border-indigo-500 transition"
                />
                <button
                    @click="handleSend"
                    class="bg-indigo-600 hover:bg-indigo-500 text-white px-6 py-3 rounded-xl font-bold transition shadow-lg active:transform active:scale-95"
                >
                    SEND
                </button>
            </div>
        </footer>
    </div>
</template>
