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

## 产品逻辑
- 核心对象是 `TradeJournal`：一条交易/观察会话，包含 `open` / `done` 状态和事件链。
- 快速记录事件只有 `entry | add | reduce | exit | log`；允许的事件类型由 journal 状态和当前净仓位共同决定。
- `reduce` 如果刚好把净仓位归零，会被领域层自动转换成 `exit`；`done` journal 只允许继续追加 `log`。
- Panel 打开时，如果当前没有 open journal 且存在传入预设或默认预设，会自动创建 journal；不要在别处复制这套初始化逻辑。
- 设置页负责保存交易对预设、默认预设、提醒开关和提醒间隔；这些都持久化在 `trajon:trading-panel-state`。
- Panel 的窗口尺寸和位置单独持久化在 `trading-panel-bounds`。
- 日历页、设置页、Panel 都读写同一份本地状态；当前没有后端持久化、同步服务或数据库读写链路。
- 图表页通过外部 script 嵌入 TradingView widget；分析页和 CSV 导出目前仍是占位/未完成状态。
