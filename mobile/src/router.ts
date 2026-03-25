import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@shared/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('./views/auth/Login.vue'),
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('./views/auth/Register.vue'),
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('./views/auth/ForgotPassword.vue'),
    },
    {
      path: '/legal/terms',
      name: 'legal-terms',
      component: () => import('./views/legal/TermsOfService.vue'),
    },
    {
      path: '/legal/privacy',
      name: 'legal-privacy',
      component: () => import('./views/legal/PrivacyPolicy.vue'),
    },
    {
      path: '/',
      component: () => import('./components/MobileLayout.vue'),
      children: [
        { path: '', redirect: '/community' },
        {
          path: 'community',
          name: 'community',
          component: () => import('./views/Community.vue'),
        },
        {
          path: 'stories',
          name: 'stories',
          component: () => import('./views/Stories.vue'),
        },
        {
          path: 'market',
          name: 'market',
          component: () => import('./views/Market.vue'),
        },
        {
          path: 'guild',
          name: 'guild',
          component: () => import('./views/Guild.vue'),
        },
        {
          path: 'guild/:id',
          name: 'guild-detail',
          component: () => import('./views/details/GuildDetail.vue'),
        },
        {
          path: 'guild/:id/posts',
          name: 'guild-posts',
          component: () => import('./views/guild/GuildPosts.vue'),
        },
        {
          path: 'guild/:id/stories',
          name: 'guild-stories',
          component: () => import('./views/guild/GuildStories.vue'),
        },
        {
          path: 'profiles',
          name: 'profiles',
          component: () => import('./views/Profiles.vue'),
        },
        {
          path: 'profile',
          name: 'profile',
          component: () => import('./views/Profile.vue'),
        },
      ],
    },
    {
      path: '/posts/:id',
      name: 'post-detail',
      component: () => import('./views/details/PostDetail.vue'),
    },
    {
      path: '/stories/:id',
      name: 'story-detail',
      component: () => import('./views/details/StoryDetail.vue'),
    },
    {
      path: '/items/:id',
      name: 'item-detail',
      component: () => import('./views/details/ItemDetail.vue'),
    },
    {
      path: '/profiles/:id',
      name: 'profile-detail',
      component: () => import('./views/details/ProfileDetail.vue'),
    },
    {
      path: '/my-favorites',
      name: 'my-favorites',
      component: () => import('./views/profile/MyFavorites.vue'),
    },
    {
      path: '/my-posts',
      name: 'my-posts',
      component: () => import('./views/profile/MyPosts.vue'),
    },
    {
      path: '/my-items',
      name: 'my-items',
      component: () => import('./views/profile/MyItems.vue'),
    },
    {
      path: '/about',
      name: 'about',
      component: () => import('./views/profile/About.vue'),
    },
  ],
})

router.beforeEach((to) => {
  const userStore = useUserStore()
  const publicPages = ['login', 'register', 'forgot-password', 'legal-terms', 'legal-privacy']
  if (!publicPages.includes(to.name as string) && !userStore.token) {
    return { name: 'login' }
  }
})

export default router
