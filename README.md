# HiJson

一款基于 [Wails](https://wails.io/) 构建的轻量级 JSON 工具桌面应用，支持 JSON 格式化、压缩、过滤、深度解析等常用操作。

## 功能特性

- **JSON 格式化** — 保持键顺序的缩进格式化
- **排序格式化** — 按键名排序后格式化输出
- **JSON 压缩** — 移除空白字符，生成紧凑 JSON
- **过滤空值** — 移除 `null`、空字符串、空对象/数组（保持键顺序）
- **深度解析** — 自动展开嵌套的 JSON 字符串
- **去换行 / 去反斜杠 / 反转义** — 常用文本清洗工具
- **文件拖拽打开** — 支持将 JSON 文件拖放到应用图标或窗口打开
- **文件读写** — 通过系统对话框打开和保存 JSON 文件

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.24 |
| 桌面框架 | Wails v2 |
| 前端 | HTML / JS（嵌入式） |

## 快速开始

### 前置条件

- [Go](https://go.dev/) ≥ 1.24
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2

### 开发模式

```bash
wails dev
```

### 构建

```bash
wails build
```

构建产物位于 `build/bin/` 目录。

## 项目结构

```
├── main.go          # 应用入口
├── app.go           # 后端 JSON 处理逻辑
├── wails.json       # Wails 项目配置
├── frontend/        # 前端资源
│   └── dist/        # 嵌入的前端构建产物
└── build/
    ├── bin/         # 可执行文件输出目录
    └── windows/     # Windows 平台资源（图标、清单等）
```

## 许可证

Copyright © 2026 Lynn

