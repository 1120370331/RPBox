import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@shared/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: () => import('./layouts/MainLayout.vue'),
      children: [
        {
          path: '',
          name: 'home',
          component: () => import('./views/HomePage.vue'),
          meta: { public: true }
        },
        {
          path: 'download',
          name: 'download',
          component: () => import('./views/DownloadPage.vue'),
          meta: { public: true }
        },
        {
          path: 'community',
          name: 'community',
          component: () => import('./views/CommunityPage.vue')
        },
        {
          path: 'community/post/:id',
          name: 'post-detail',
          component: () => import('./views/PostDetailPage.vue')
        }
      ]
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('./views/LoginPage.vue'),
      meta: { public: true }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('./views/RegisterPage.vue'),
      meta: { public: true }
    }
  ]
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()

  if (!to.meta.public && !userStore.token) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else {
    next()
  }
})

export default router
