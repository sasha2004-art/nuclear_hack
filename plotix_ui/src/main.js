import { createApp } from 'vue';
import { createPinia } from 'pinia';
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate';
import { createRouter, createWebHashHistory } from 'vue-router';
import App from './App.vue';
import './style.css';

const pinia = createPinia();
pinia.use(piniaPluginPersistedstate);

const router = createRouter({
    history: createWebHashHistory(),
    routes: [
        { path: '/', component: () => import('./views/HomeView.vue') },
        { path: '/chat/:id', component: () => import('./views/ChatView.vue') }
    ]
});

const app = createApp(App);
app.use(pinia);
app.use(router);
app.mount('#app');
