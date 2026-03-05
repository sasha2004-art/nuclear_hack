<script setup>
import { ref, computed, nextTick, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useChatStore } from '../stores/chat';

const route = useRoute();
const store = useChatStore();
const peerId = computed(() => decodeURIComponent(route.params.id));
const messages = computed(() => store.messages[peerId.value] || []);

const messageText = ref('');
const scrollContainer = ref(null);

const scrollToBottom = async () => {
    await nextTick();
    if (scrollContainer.value) {
        scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight;
    }
};

watch(messages, () => scrollToBottom(), { deep: true });
watch(peerId, () => scrollToBottom());

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
            <div>
                <h2 class="font-bold text-gray-200">{{ peerId }}</h2>
                <p class="text-xs text-gray-500">Прямое TCP соединение</p>
            </div>
        </header>

        
        <div ref="scrollContainer" class="flex-1 overflow-y-auto p-6 space-y-4 bg-gray-900">
            <div v-for="(msg, idx) in messages" :key="idx"
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

        
/div>
        </div>

        
s="max-w-4xl mx-auto flex gap-4">
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
                    ОТПРАВИТЬ
                </button>
                    SEND
ooter>
    </div>
</template>
