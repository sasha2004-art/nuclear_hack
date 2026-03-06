import { defineStore } from 'pinia';
import axios from 'axios';

const API_URL = 'http://localhost:8080';
const WS_URL = 'ws://localhost:8080/events';

export const useChatStore = defineStore('chat', {
    state: () => ({
        peers: {},
        messages: {},
        myId: 'unknown',
        wsConnected: false,
        accounts: [],
        activeAccountId: ''
    }),

    persist: true,

    actions: {
        async fetchMe() {
            try {
                const res = await axios.get(`${API_URL}/me`);
                this.myId = res.data.peer_id;
                this.activeAccountId = res.data.peer_id;
            } catch (e) {
                console.error('Failed to fetch identity');
            }
        },

        async fetchAccounts() {
            try {
                const res = await axios.get(`${API_URL}/accounts`);
                this.accounts = res.data.accounts || [];
                this.activeAccountId = res.data.active_id || '';
            } catch (e) {
                console.error('Failed to fetch accounts');
            }
        },

        async createAccount(name) {
            try {
                const res = await axios.post(`${API_URL}/accounts/create`, { name });
                await this.fetchAccounts();
                return res.data;
            } catch (e) {
                console.error('Failed to create account');
                return null;
            }
        },

        async switchAccount(peerId) {
            try {
                await axios.post(`${API_URL}/accounts/switch`, { peer_id: peerId });
                this.messages = {};
                this.peers = {};
                await this.fetchMe();
                await this.fetchAccounts();
                await this.fetchPeers();
            } catch (e) {
                console.error('Failed to switch account');
            }
        },

        async renameAccount(peerId, name) {
            try {
                await axios.post(`${API_URL}/accounts/rename`, { peer_id: peerId, name });
                await this.fetchAccounts();
            } catch (e) {
                console.error('Failed to rename account');
            }
        },

        async setGhost(peerId, ghost) {
            try {
                await axios.post(`${API_URL}/accounts/ghost`, { peer_id: peerId, ghost });
                await this.fetchAccounts();
            } catch (e) {
                console.error('Failed to set ghost mode');
            }
        },

        async renamePeerLocal(peerId, name) {
            try {
                await axios.post(`${API_URL}/peer/rename`, { peer_id: peerId, name });
                if (this.peers[peerId]) {
                    this.peers[peerId].name = name;
                }
                await this.fetchPeers();
            } catch (e) {
                console.error('Failed to rename peer locally');
            }
        },

        async fetchPeers() {
            try {
                const res = await axios.get(`${API_URL}/peers`);
                this.peers = res.data;
            } catch (e) {
                console.error('Failed to fetch peers');
            }
        },

        async sendMessage(peerId, text) {
            try {
                await axios.post(`${API_URL}/send_message`, {
                    peer_id: peerId,
                    message: text
                });
                this.pushMessage(peerId, text, true);
                return true;
            } catch (e) {
                return false;
            }
        },

        pushMessage(peerId, text, isSelf) {
            if (!this.messages[peerId]) {
                this.messages[peerId] = [];
            }
            this.messages[peerId].push({
                text,
                self: isSelf,
                timestamp: Date.now()
            });
            this.messages[peerId].sort((a, b) => a.timestamp - b.timestamp);
            this.messages = { ...this.messages };
        },

        async fetchHistory(peerId) {
            try {
                const res = await axios.get(`${API_URL}/history?peer_id=${peerId}`);
                if (res.data && res.data.length > 0) {
                    res.data.sort((a, b) => a.timestamp - b.timestamp);
                    this.messages[peerId] = res.data.map(m => ({
                        text: m.text,
                        self: m.sender === this.myId,
                        timestamp: m.timestamp,
                        delivered: m.delivered
                    }));
                    this.messages = { ...this.messages };
                }
            } catch (e) {
                console.error('Failed to fetch history');
            }
        },

        initWebSocket() {
            if (this._socket) {
                this._socket.close();
            }

            const socket = new WebSocket(WS_URL);
            this._socket = socket;

            socket.onopen = () => {
                this.wsConnected = true;
            };

            socket.onmessage = (event) => {
                const data = JSON.parse(event.data);
                if (data.type === 'new_message') {
                    this.pushMessage(data.payload.sender, data.payload.text, false);
                }
                if (data.type === 'account_switched') {
                    this.messages = {};
                    this.peers = {};
                    this.fetchMe();
                    this.fetchAccounts();
                    this.fetchPeers();
                }
            };

            socket.onclose = () => {
                this.wsConnected = false;
                setTimeout(() => this.initWebSocket(), 3000);
            };
        }
    }
});
