# AGENTS.md

## 项目定位
- 这是单仓库单应用，不是 monorepo：前端在 `src/`，Tauri 壳层在 `src-tauri/`。
- 根目录 `README.md` 和 `index.html` 仍是 Tauri/Vite 模板文案；以 `package.json`、`vite.config.ts`、`src-tauri/tauri.conf.json` 和 `src/` 实际代码为准。

## 技术栈
- 前端：Vite 7 + React 19 + TypeScript strict + Ant Design 6。
- 桌面壳：Tauri 2 + Rust。Rust 端当前只负责注册插件（`fs`、`opener`、`sql`）和模板 `greet` 命令，业务逻辑基本都在前端。
- 测试：Vitest 3 + jsdom + Testing Library；`src/test/setup.ts` 预置了 `ResizeObserver` 和 `matchMedia` mock。

## 常用命令
- 优先使用 `bun`。根目录存在 `bun.lock`，且 `src-tauri/tauri.conf.json` 的 `beforeDevCommand` / `beforeBuildCommand` 明确写死为 `bun run dev` / `bun run build`。
- 前端开发：`bun run dev`
- 前端构建：`bun run build`
- 全量测试：`bun run test`
- 跑单个测试文件：`bun run test -- src/tradingPanelDomain.test.ts`
- Tauri 桌面开发：`bun run tauri dev`
- 仓库当前没有 `lint` / `format` script，也没有可读到的 CI workflow；不要臆造 `bun run lint` 之类的命令。

## 架构
- `src/main.tsx` 只挂载 React 和 AntD theme；真正的入口分流在 `src/App.tsx`。
- 这个项目没有 React Router；`src/App.tsx` 通过 `parseAppView(window.location.search)` 用 URL 参数切换主界面和浮动 panel。
- 主界面在 `src/components/tradingPanel/TradingPanelMain.tsx`，浮动快速记录面板在 `src/components/tradingPanel/TradingPanelPanel.tsx`。
- `src/tradingPanelStore.ts` 是唯一状态入口：读取、规范化、写回 `localStorage`，并暴露所有 state actions。改产品行为先看这里，再改 UI。
- `src/tradingPanelDomain.ts` 集中承载交易规则：事件类型、净仓位、校验、日历分组都在这里；不要把规则散回组件层。

## 产品逻辑
- 核心对象是 `TradeJournal`：一条交易/观察会话，包含 `open` / `done` 状态和事件链。
- 快速记录事件只有 `entry | add | reduce | exit | log`；允许的事件类型由 journal 状态和当前净仓位共同决定。
- `reduce` 如果刚好把净仓位归零，会被领域层自动转换成 `exit`；`done` journal 只允许继续追加 `log`。
- Panel 打开时，如果当前没有 open journal 且存在传入预设或默认预设，会自动创建 journal；不要在别处复制这套初始化逻辑。
- 设置页负责保存交易对预设、默认预设、提醒开关和提醒间隔；这些都持久化在 `trading-flow:trading-panel-state`。
- Panel 的窗口尺寸和位置单独持久化在 `trading-panel-bounds`。
- 日历页、设置页、Panel 都读写同一份本地状态；当前没有后端持久化、同步服务或数据库读写链路。
- 图表页通过外部 script 嵌入 TradingView widget；分析页和 CSV 导出目前仍是占位/未完成状态。

## Tauri / Window 注意点
- `src/panelWindow.ts` 在 Tauri 环境创建 label 为 `trading-panel` 的独立 webview；在浏览器或测试环境会退化成 `window.open` popup。
- 调用窗口 API 前要保留现有的 runtime 判断和 browser fallback，否则 Vitest/jsdom 环境很容易报错。
- `panelPinned` 会同步到 Tauri 窗口置顶状态，但这个行为只在 Tauri runtime 下生效。

## 测试与改动落点
- 交易规则相关改动优先补 `src/tradingPanelDomain.test.ts`；状态持久化或 hook 行为优先补 `src/tradingPanelStore.test.tsx`。
- 入口/主题相关回归看 `src/main.test.tsx`；窗口行为看 `src/panelWindow.test.ts`。
- `src-tauri/target/` 和 `src-tauri/gen/schemas/` 是构建产物/生成物，不要把它们当源码编辑或总结。
