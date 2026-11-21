# Xin Skill Vault · Frontend

Vue 3 + TypeScript + Vite + Pinia + Vue Router。

```bash
npm install
npm run dev        # http://127.0.0.1:9245
npm run build      # vue-tsc + vite build
```

浏览器预览时没有 Wails 后端，`src/lib/api.ts` 会自动使用 demo 配置；桌面开发请从仓库根目录运行 `wails3 task dev`。

绑定通过以下命令生成（不要手改 `bindings/`）：

```bash
wails3 generate bindings -ts -i -d frontend/bindings ./...
```
