import type { RouteRecordRaw } from 'vue-router';

const qrRoutes: RouteRecordRaw[] = [
  {
    path: '/q/:token',
    name: 'label-qr-scan',
    component: () => import('~/pages/label/QRScanPage.vue'),
    // 扫码页受全局登录守卫保护；登录完成后原始 URL 会自动回跳继续解析。
    meta: { public: false, title: '二维码定位' },
  },
];

export default qrRoutes;
