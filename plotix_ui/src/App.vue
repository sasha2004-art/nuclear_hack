<script setup>
import { onMounted } from 'vue';
import { useChatStore } from './stores/chat';

const store = useChatStore();

onMounted(() => {
    store.fetchPeers();
    store.initWebSocket();
    setInterval(() => store.fetchPeers(), 5000);
});
</script>

<template>
    <div class="flex h-screen bg-gray-900 text-gray-100 font-sans">
        
        <aside class="w-80 bg-gray-800 border-r border-gray-700 flex flex-col">
            <div class="p-6 border-b border-gray-700">
                <h1 class="text-xl font-bold text-indigo-400">Plotix Local</h1>
                <div class="flex items-center mt-2">
                    <div :class="['w-2 h-2 rounded-full mr-2', store.wsConnected ? 'bg-green-500' : 'bg-red-500']"></div>
                    <span class="text-xs text-gray-400">Ядро: {{ store.wsConnected ? 'В СЕТИ' : 'НЕ В СЕТИ' }}</span>
                </div>
            </div>

            <nav class="flex-1 overflow-y-auto">
                <div v-if="Object.keys(store.peers).length === 0" class="p-10 text-center text-gray-500 text-sm">
                    Поиск участников сети...
                </div>
                <router-link
                    v-for="(ip, id) in store.peers"
                    :key="id"
                    :to="'/chat/' + encodeURIComponent(id)"
                    class="block p-4 border-b border-gray-700/50 hover:bg-gray-700 transition"
                    active-class="bg-gray-700 border-l-4 border-indigo-500"
                >
                    <div class="font-medium truncate text-sm">{{ id }}</div>
                    <div class="text-xs text-gray-500 mt-1">{{ ip }}</div>
                </router-link>
            </nav>
        </aside>

        
        <main class="flex-1 flex flex-col overflow-hidden">
            <router-view></router-view>
        </main>
    </div>
</template>
