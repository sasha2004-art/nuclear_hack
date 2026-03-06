<script setup>
import { onMounted, ref } from 'vue';
import { useChatStore } from './stores/chat';

const store = useChatStore();
const showAccountMenu = ref(false);
const renamingId = ref(null);
const renameText = ref('');

onMounted(async () => {
    await store.fetchMe();
    await store.fetchAccounts();
    store.fetchPeers();
    store.initWebSocket();
    setInterval(() => store.fetchPeers(), 5000);
});

const truncateId = (id) => {
    if (!id) return '';
    return id.length > 16 ? id.slice(0, 16) + '...' : id;
};

const getAccountLabel = (acc) => {
    return acc.name || truncateId(acc.peer_id);
};

const getInitial = (acc) => {
    if (acc.name) return acc.name.charAt(0).toUpperCase();
    return acc.peer_id ? acc.peer_id.replace('hex_', '').charAt(0).toUpperCase() : '?';
};

const avatarColors = [
    'bg-indigo-600', 'bg-emerald-600', 'bg-rose-600', 'bg-amber-600',
    'bg-cyan-600', 'bg-purple-600', 'bg-pink-600', 'bg-teal-600'
];

const getAvatarColor = (peerId) => {
    let hash = 0;
    for (let i = 0; i < peerId.length; i++) hash = peerId.charCodeAt(i) + ((hash << 5) - hash);
    return avatarColors[Math.abs(hash) % avatarColors.length];
};

const currentAccount = () => {
    return store.accounts.find(a => a.peer_id === store.activeAccountId);
};

const handleSwitch = async (peerId) => {
    if (peerId === store.activeAccountId) return;
    await store.switchAccount(peerId);
    showAccountMenu.value = false;
};

const handleCreate = async () => {
    await store.createAccount('');
    await store.fetchAccounts();
};

const startRename = (acc) => {
    renamingId.value = acc.peer_id;
    renameText.value = acc.name || '';
};

const confirmRename = async () => {
    if (renamingId.value) {
        await store.renameAccount(renamingId.value, renameText.value);
        renamingId.value = null;
        renameText.value = '';
    }
};
</script>

<template>
    <div class="flex h-screen bg-gray-900 text-gray-100 font-sans">

        <aside class="w-80 bg-gray-800 border-r border-gray-700 flex flex-col">
            
            <div class="p-4 border-b border-gray-700">
                <div class="flex items-center justify-between">
                    <h1 class="text-lg font-bold text-indigo-400">Plotix Local</h1>
                </div>
                <div
                    class="flex items-center mt-3 cursor-pointer hover:bg-gray-700/50 rounded-lg p-2 -mx-2 transition"
                    @click="showAccountMenu = !showAccountMenu"
                >
                    <div
                        :class="[
                            'w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-bold shrink-0',
                            currentAccount() ? getAvatarColor(store.activeAccountId) : 'bg-gray-600'
                        ]"
                    >
                        {{ currentAccount() ? getInitial(currentAccount()) : '?' }}
                    </div>
                    <div class="ml-3 min-w-0 flex-1">
                        <div class="text-sm font-medium truncate">{{ currentAccount() ? getAccountLabel(currentAccount()) : 'No account' }}</div>
                        <div class="flex items-center">
                            <div :class="['w-1.5 h-1.5 rounded-full mr-1.5', store.wsConnected ? 'bg-green-500' : 'bg-red-500']"></div>
                            <span class="text-[10px] text-gray-500">{{ store.wsConnected ? 'Online' : 'Offline' }}</span>
                        </div>
                    </div>
                    <svg class="w-4 h-4 text-gray-500 shrink-0 transition-transform" :class="{ 'rotate-180': showAccountMenu }" viewBox="0 0 20 20" fill="currentColor">
                        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
                    </svg>
                </div>
            </div>

            
            <div v-if="showAccountMenu" class="border-b border-gray-700">
                <div
                    v-for="acc in store.accounts"
                    :key="acc.peer_id"
                    class="flex items-center p-3 hover:bg-gray-700 transition cursor-pointer border-b border-gray-700/30"
                    :class="{ 'bg-gray-700/50 border-l-2 border-indigo-500': acc.peer_id === store.activeAccountId }"
                    @click="handleSwitch(acc.peer_id)"
                >
                    
                    <div
                        :class="[
                            'w-7 h-7 rounded-full flex items-center justify-center text-white text-xs font-bold shrink-0',
                            getAvatarColor(acc.peer_id)
                        ]"
                    >
                        {{ getInitial(acc) }}
                    </div>

                    
                    <div class="ml-3 flex-1 min-w-0">
                        
                        <div v-if="renamingId === acc.peer_id" class="flex gap-2" @click.stop>
                            <input
                                v-model="renameText"
                                @keyup.enter="confirmRename"
                                class="flex-1 bg-gray-900 border border-gray-600 rounded px-2 py-1 text-xs text-gray-100 focus:outline-none focus:border-indigo-500"
                                placeholder="Nickname..."
                                autofocus
                            />
                            <button @click="confirmRename" class="text-xs text-green-400 hover:text-green-300">OK</button>
                        </div>
                        
                        <div v-else>
                            <div class="text-sm font-medium truncate">{{ getAccountLabel(acc) }}</div>
                            <div class="text-[10px] text-gray-500 truncate">{{ truncateId(acc.peer_id) }}</div>
                        </div>
                    </div>

                    
                    <div class="flex items-center gap-2 ml-2 shrink-0">
                        
                        <button
                            v-if="acc.peer_id === store.activeAccountId"
                            @click.stop="store.setGhost(acc.peer_id, !acc.ghost)"
                            :class="['text-xs px-1.5 py-0.5 rounded transition', acc.ghost ? 'bg-gray-600 text-gray-300' : 'bg-transparent text-gray-500 hover:text-gray-300']"
                            :title="acc.ghost ? 'Ghost ON' : 'Ghost OFF'"
                        >
                            {{ acc.ghost ? '&#128123;' : '&#128065;' }}
                        </button>
                        
                        <button
                            v-if="renamingId !== acc.peer_id"
                            @click.stop="startRename(acc)"
                            class="text-gray-500 hover:text-gray-300 text-xs"
                            title="Rename"
                        >
                            &#9998;
                        </button>
                    </div>
                </div>

                <button
                    @click="handleCreate"
                    class="w-full p-3 text-sm text-indigo-400 hover:bg-gray-700 transition text-center font-medium flex items-center justify-center gap-2"
                >
                    <div class="w-7 h-7 rounded-full border-2 border-dashed border-indigo-500/50 flex items-center justify-center text-indigo-400 text-sm">+</div>
                    New Account
                </button>
            </div>

            <nav class="flex-1 overflow-y-auto">
                <div v-if="Object.keys(store.peers).length === 0" class="p-10 text-center text-gray-500 text-sm">
                    No chats yet...
                </div>
                <router-link
                    v-for="peer in Object.entries(store.peers).sort((a, b) => Number(b[1].online) - Number(a[1].online))"
                    :key="peer[0]"
                    :to="'/chat/' + encodeURIComponent(peer[0])"
                    class="flex items-center p-4 border-b border-gray-700/50 hover:bg-gray-700 transition relative group"
                    active-class="bg-gray-700 border-l-4 border-indigo-500"
                >
                    <div class="relative">
                        <div
                            :class="[
                                'w-10 h-10 rounded-full flex items-center justify-center text-white text-sm font-bold shrink-0',
                                getAvatarColor(peer[0]),
                                !peer[1].online ? 'opacity-50 grayscale' : ''
                            ]"
                        >
                            {{ peer[1].name ? peer[1].name.charAt(0).toUpperCase() : peer[0].replace('hex_', '').charAt(0).toUpperCase() }}
                        </div>
                        <div
                            class="absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-gray-800"
                            :class="peer[1].online ? 'bg-green-500' : 'bg-gray-500'"
                            :title="peer[1].online ? 'Online' : 'Offline'"
                        ></div>
                    </div>
                    <div class="ml-3 min-w-0 flex-1">
                        <div class="flex justify-between items-baseline">
                            <div class="font-medium truncate text-sm text-gray-200">
                                {{ peer[1].name || truncateId(peer[0]) }}
                            </div>
                        </div>
                        <div class="text-xs text-gray-500 truncate mt-0.5">
                            {{ peer[1].online ? 'Active' : 'Offline' }}
                        </div>
                    </div>
                </router-link>
            </nav>
        </aside>

        <main class="flex-1 flex flex-col overflow-hidden">
            <router-view></router-view>
        </main>
    </div>
</template>
