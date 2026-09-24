# 灵衍云移动端

移动宿主使用 Vue 3、TypeScript、Vite、Pinia、Vue Router 与 Vant 4。桌面端和移动端
共享表单 Schema、Runtime Core、规则、校验及提交协议，移动端只维护独立的页面外壳和
Vant 字段呈现层。

## 本地运行

在 `apps/evolyn-web` 目录执行：

```bash
pnpm --filter @evolyn.do/mobile dev
```

默认地址为 `http://localhost:11001`，`/api` 会代理到本地 `8080` 端口的后端。

## 运行时入口

移动业务页统一从 `@evolyn.do/form/runtime-mobile` 引入 `FormMobileRuntimeSurface`，并由
宿主注入 `FormRuntimeAdapter`。禁止移动页面直接引用 `runtime-web` 或 Element Plus。
宿主入口需先加载 `vant/lib/index.css`，再加载
`@evolyn.do/form/runtime-mobile/style.css`，使品牌变量可以统一覆盖两者。

`/runtime-preview` 是不依赖后端的验收页，用于持续验证移动字段、校验、确认框与安全区
操作栏；正式表单页面应加载服务端发布的不可变 Schema，并用真实 API adapter 替换预览
adapter。
