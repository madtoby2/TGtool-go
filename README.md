# ⚡ 疾风TG营销助手 v2.0 (Go)

Telegram 营销自动化 — 纯 Go 实现，单文件可执行。

基于 [gotd/td](https://github.com/gotd/td) MTProto 协议库，编译为单一二进制文件，无需 Python 运行时。

## 安装

### 下载预编译

```bash
# Windows
tgtool.exe setup

# Linux / macOS
./tgtool setup
```

### 从源码编译

```bash
git clone https://github.com/madtoby2/TGtool.git
cd TGtool/go
make build          # Windows: tgtool.exe
make build-linux    # Linux: tgtool
make build-mac      # macOS: tgtool

# 制作安装程序 (需要 NSIS)
make installer      # → tgtool-setup-2.0.0.exe
```

## 命令

| 命令 | 说明 |
|------|------|
| `setup` | 首次配置 api_id/api_hash |
| `login` | 登录账号 |
| `accounts` | 列出所有账号 |
| `collect-query <关键词>` | 关键词采集群组 |
| `collect-members <群链接>` | 采集群成员 |
| `join <群文件>` | 批量加群 |
| `send <消息文件>` | 群组群发 |
| `dm <目标> <消息>` | 私信群发 |
| `invite <用户> <群>` | 批量邀请 |
| `farm <话术> <群>` | 炒群 |
| `filter <手机号文件>` | 筛号 |
| `clone <源> <目标>` | 频道复刻 |
| `config` | 查看配置 |

## 项目结构

```
├── cmd/tgtool/main.go       # CLI 入口
├── internal/
│   ├── config/config.go     # 配置管理
│   ├── session/session.go   # 登录/session管理
│   ├── collector/collector.go  # 批量采集
│   ├── sender/sender.go     # 群发/加群/邀请/DM
│   ├── filter/filter.go     # 手机筛号
│   ├── farming/farming.go   # 炒群/话术采集
│   └── clone/clone.go       # 频道复刻
├── installer.nsi            # NSIS 安装脚本
├── Makefile
└── go.mod
```

## 特性

- 单一二进制 (~20MB)，免安装 Python 运行时
- MTProto 协议，非 Bot API
- 跨平台编译 (Windows/Linux/macOS)
- 多账号并发 + 随机间隔防封

## 免责声明

本工具仅供学习交流，使用者自行承担一切法律责任。
