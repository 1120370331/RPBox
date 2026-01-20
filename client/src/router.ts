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
        { path: 'sync/profile/:id/edit', name: 'profile-edit', component: () => import('./views/sync/ProfileEdit.vue') },
        { path: 'archives', name: 'archives', component: () => import('./views/archives/ArchivesMain.vue') },
        { path: 'archives/story/:id', name: 'story-detail', component: () => import('./views/archives/StoryDetail.vue') },
        { path: 'market', name: 'market', component: () => import('./views/market/MarketMain.vue') },
        { path: 'market/upload', name: 'item-upload', component: () => import('./views/market/ItemUpload.vue') },
        { path: 'market/my-items', name: 'my-items', component: () => import('./views/market/MyItems.vue') },
        { path: 'market/preview', name: 'item-preview', component: () => import('./views/market/ItemPreview.vue') },
        { path: 'market/:id', name: 'item-detail', component: () => import('./views/market/ItemDetail.vue') },
        { path: 'market/:id/edit', name: 'item-edit', component: () => import('./views/market/ItemEdit.vue') },
        { path: 'community', name: 'community', component: () => import('./views/community/CommunityMain.vue') },
        { path: 'community/create', name: 'post-create', component: () => import('./views/community/PostCreate.vue') },
        { path: 'community/my-posts', name: 'my-posts', component: () => import('./views/community/MyPosts.vue') },
        { path: 'community/post/:id', name: 'post-detail', component: () => import('./views/community/PostDetail.vue') },
        { path: 'community/post/:id/edit', name: 'post-edit', component: () => import('./views/community/PostEdit.vue') },
        { path: 'community/preview', name: 'post-preview', component: () => import('./views/community/PostPreview.vue') },
        { path: 'library/favorites', name: 'library-favorites', component: () => import('./views/library/Favorites.vue') },
        { path: 'library/history', name: 'library-history', component: () => import('./views/library/History.vue') },
        { path: 'notifications', name: 'notifications', component: () => import('./views/Notifications.vue') },
        { path: 'settings', name: 'settings', component: () => import('./views/Settings.vue') },
        { path: 'guide', name: 'guide', component: () => import('./views/Guide.vue') },
        { path: 'guild', name: 'guild', component: () => import('./views/guild/GuildList.vue') },
        { path: 'guild/create', name: 'guild-create', component: () => import('./views/guild/GuildCreate.vue') },
        { path: 'guild/:id', name: 'guild-detail', component: () => import('./views/guild/GuildDetail.vue') },
        { path: 'guild/:id/manage', name: 'guild-manage', component: () => import('./views/guild/GuildManage.vue') },
        { path: 'guild/:id/posts', name: 'guild-posts', component: () => import('./views/guild/GuildPosts.vue') },
        { path: 'guild/:id/stories', name: 'guild-stories', component: () => import('./views/guild/GuildStories.vue') },
        // 版主中心
        { path: 'moderator', name: 'moderator', component: () => import('./views/moderator/ModeratorMain.vue') },
        // 用户主页
        { path: 'user/:id', name: 'user-profile', component: () => import('./views/user/UserProfile.vue') },
      ]
    },
    { path: '/login', name: 'login', component: () => import('./views/Login.vue') },
    { path: '/register', name: 'register', component: () => import('./views/Register.vue') },
    { path: '/forgot-password', name: 'forgot-password', component: () => import('./views/ForgotPassword.vue') },
    { path: '/story/:code', name: 'story-playback', component: () => import('./views/archives/StoryPlayback.vue') },
  ],
})

export default router
