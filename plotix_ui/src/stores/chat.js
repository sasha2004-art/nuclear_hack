import { defineStore } from 'pinia';
import axios from 'axios';

const API_URL = 'http://localhost:8080';
const WS_URL = 'ws://localhost:8080/events';

export const useChatStore = defineStore('chat', {
    state: () => ({
        peers: {},
        messages: {},
        myId: 'unknown',
        wsConnected: false
    }),

    persist: true,

    actions: {
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
            this.messages = { ...this.messages };
        },

        initWebSocket() {
            const socket = new WebSocket(WS_URL);

            socket.onopen = () => {
                this.wsConnected = true;
            };

            socket.onmessage = (event) => {
                const data = JSON.parse(event.data);
                if (data.type === 'new_message') {
                    this.pushMessage(data.payload.sender, data.payload.text, false);
                }
            };

            socket.onclose = () => {
                this.wsConnected = false;
                setTimeout(() => this.initWebSocket(), 3000);
            };
        }
    }
});
