import Layout from '@/layout';
import ErrorPage from '@/pages/error';
import HomePage from '@/pages/home';
import Login from '@/pages/login';
import ProductPage from '@/pages/product';
import ShowcasePage from '@/pages/showcase';
import { createRootRoute, createRoute, createRouter, Outlet } from '@tanstack/react-router';

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

const homeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: HomePage,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: Login,
});

const showcaseRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/showcase',
  component: ShowcasePage,
});

const productRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/product',
  component: ProductPage,
});

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'dashboard',
  component: Layout,
});

const dashboardIndexRoute = createRoute({
  getParentRoute: () => dashboardRoute,
  path: '/dashboard',
  component: HomePage,
});

const errorRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '$',
  component: ErrorPage,
});

const routeTree = rootRoute.addChildren([
  homeRoute,
  loginRoute,
  showcaseRoute,
  productRoute,
  dashboardRoute.addChildren([dashboardIndexRoute]),
  errorRoute,
]);

export const router = createRouter({ routeTree });

// 声明路由类型（TanStack Router 类型安全需要）
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
