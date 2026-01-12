import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from './components/AppLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: '', name: 'home', component: () => import('./views/Home.vue') },
        { path: 'sync', name: 'sync', component: () => import('./views/sync/SyncMain.vue') },
        { path: 'sync/setup', name: 'sync-setup', component: () => import('./views/sync/SetupWizard.vue') },
        { path: 'sync/profile/:id', name: 'profile-detail', component: () => import('./views/sync/ProfileDetail.vue') },
        { path: 'settings', name: 'settings', component: () => import('./views/Settings.vue') },
      ]
    },
    { path: '/login', name: 'login', component: () => import('./views/Login.vue') },
    { path: '/register', name: 'register', component: () => import('./views/Register.vue') },
  ],
})

export default router
