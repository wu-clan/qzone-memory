# QQ 空间回忆

一个用 Go 语言开发的 QQ 空间回忆项目

## 快速开始

### 1. 安装依赖

```bash
make deps
```

### 2. 运行

```bash
# 编译并运行
make run
```

## 数据与隐私

- **数据全程在本地**：登录态、数据库（`data/qzone.db`）与全部图片（`data/media/<qq>/…`）只保存在你这台电脑，不经任何服务器、无埋点、无上报。
- **图片永久保存**：同步会在最后一步把照片 / 头像下载到本地；下载完成后断网也能完整浏览。
- **抓取克制**：所有对腾讯的请求经统一限速，避免触发风控。
- **视频原片开关**：默认只存视频封面；如需连同视频原片一起下载，将 `media.download_videos` 设为 `true`。
- **数据归我**：界面「数据与隐私」面板可查看存储位置与媒体占用、重新下载失败项，或**彻底删除**本机上该账号的全部数据。

## 配置

主要配置位于 `config/config.yaml`：

| 配置项 | 说明 | 默认 |
|---|---|---|
| `qzone.request_interval_ms` | 全局最小请求间隔（毫秒），抓取与下载共用 | 800 |
| `qzone.max_concurrency` | 出站最大并发 | 3 |
| `qzone.jitter_ms` | 随机抖动上限（毫秒） | 600 |
| `media.dir` | 媒体落盘目录 | `./data/media` |
| `media.download_concurrency` | 媒体下载并发 | 3 |
| `media.max_retries` | 单个媒体最大重试次数 | 3 |
| `media.download_videos` | 是否下载视频原片 | `false` |

> 找不到 `config/config.yaml` 时会使用内置默认值，因此单个可执行文件也能直接运行。

## 分发 / 打包

```bash
# 构建当前平台的发布版（输出到 dist/）
make release
```

- 运行后会**自动打开浏览器**到 `http://127.0.0.1:8081`（设置环境变量 `QZONE_NO_BROWSER=1` 可关闭）。
- 程序会把 `data/`、`logs/` 创建在**可执行文件所在目录**，因此双击运行也能正常读写。
- 跨平台构建：`make release-all`（注意本项目用 CGO 版 SQLite，跨平台编译需对应平台的 C 工具链，缺失会自动跳过；建议在各目标平台本地构建或用 CI）。
