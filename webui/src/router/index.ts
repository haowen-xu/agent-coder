import type { Pinia } from 'pinia'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { applyRouteGuards } from './guards'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    component: () => import('../layouts/AuthLayout.vue'),
    children: [
      {
        path: '',
        name: 'login',
        component: () => import('../views/LoginView.vue'),
        meta: { guestOnly: true },
      },
    ],
  },
  {
    path: '/',
    component: () => import('../layouts/ConsoleLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: { name: 'board' },
      },
      {
        path: 'board',
        name: 'board',
        component: () => import('../views/BoardView.vue'),
      },
      {
        path: 'admin',
        redirect: { name: 'admin-overview' },
        meta: { requiresAdmin: true },
      },
      {
        path: 'admin/overview',
        name: 'admin-overview',
        component: () => import('../views/admin/AdminOverviewView.vue'),
        meta: { requiresAdmin: true },
      },
      {
        path: 'admin/users',
        name: 'admin-users',
        component: () => import('../views/admin/AdminUsersView.vue'),
        meta: { requiresAdmin: true },
      },
      {
        path: 'admin/projects',
        name: 'admin-projects',
        component: () => import('../views/admin/AdminProjectsView.vue'),
        meta: { requiresAdmin: true },
      },
      {
        path: 'admin/prompts',
        name: 'admin-prompts',
        component: () => import('../views/admin/AdminPromptsView.vue'),
        meta: { requiresAdmin: true },
      },
      {
        path: 'admin/ops',
        name: 'admin-ops',
        component: () => import('../views/admin/AdminOpsView.vue'),
        meta: { requiresAdmin: true },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
]

export function createAppRouter(pinia: Pinia) {
  const router = createRouter({
    history: createWebHistory(),
    routes,
  })

  applyRouteGuards(router, pinia)

  return router
}
