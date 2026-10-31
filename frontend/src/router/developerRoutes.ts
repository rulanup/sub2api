import type { RouteRecordRaw } from 'vue-router'

export const developerRoutes: RouteRecordRaw[] = [
  {
    path: '/developer',
    name: 'DeveloperPortal',
    component: () => import('@/features/developer/DeveloperPortalView.vue'),
    redirect: '/developer/docs/getting-started',
    meta: {
      requiresAuth: false,
      title: 'Developer Portal',
      titleKey: 'developer.portalTitle',
    },
    children: [
      {
        path: 'docs/:slug?',
        name: 'DeveloperDocs',
        component: () => import('@/features/developer/views/DeveloperDocsView.vue'),
        props: true,
        meta: {
          requiresAuth: false,
          title: 'Documentation',
          titleKey: 'developer.navigation.docs',
        },
      },
      {
        path: 'api',
        name: 'DeveloperApiReference',
        component: () => import('@/features/developer/views/DeveloperApiView.vue'),
        meta: {
          requiresAuth: false,
          title: 'API Reference',
          titleKey: 'developer.navigation.api',
        },
      },
      {
        path: 'playground',
        name: 'DeveloperPlayground',
        component: () => import('@/features/developer/views/DeveloperPlaygroundView.vue'),
        meta: {
          requiresAuth: false,
          title: 'API Playground',
          titleKey: 'developer.navigation.playground',
        },
      },
      {
        path: 'workspace',
        name: 'DeveloperWorkspace',
        component: () => import('@/features/developer/views/DeveloperWorkspaceView.vue'),
        meta: {
          requiresAuth: false,
          title: 'Request Workspace',
          titleKey: 'developer.navigation.workspace',
        },
      },
      {
        path: 'debug',
        name: 'DeveloperDebug',
        component: () => import('@/features/developer/views/DeveloperDebugView.vue'),
        meta: {
          requiresAuth: false,
          title: 'Debug Center',
          titleKey: 'developer.navigation.debug',
        },
      },
    ],
  },
]
