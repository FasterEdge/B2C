# B2C 中文文档

本仓库的中文主文档为 [README.md](./README.md)。

## 关于本仓库的性质（重要）

**B2C 是 [LF Edge eKuiper](https://github.com/lf-edge/ekuiper) 的同步副本（fork）**，不是自研引擎。

- 与上游的分界点为提交 `fce9a2e1`（上游 `feat: support array_avg (#4108)`，2026-08-18）。
- 分界点至今共 **55 个自有提交**，净增 **632 行**（其中 Go 代码净增 1277 行），占上游 125418 行被跟踪代码（`git ls-files | xargs wc -l`）的约 **0.50%**。
- 本分支唯一自研功能为 **MQTT v5 请求-响应网关**（见 README 第六节），
  其余自有提交为安全修复、构建/CI 适配与文档修正。
- 因此，**凡 README 未标注「本分支新增」的能力，均来自上游 eKuiper**。使用与二次开发请
  以上游仓库的文档和社区为主要参考。

## 延伸阅读

- [README.md](./README.md) —— 中文主文档，含能力归属表与本分支新增功能详解
- [README_en.md](./README_en.md) —— 英文主文档
- [项目文档](./docs/zh_CN/) —— 上游中英文文档（本分支仅修正失效链接）
- [REST sink · 响应回传（请求-响应网关）](./docs/zh_CN/guide/sinks/builtin/rest.md) —— 自研功能完整文档
- [RELEASE_NOTES.md](./RELEASE_NOTES.md) —— 默认绑定 localhost 的行为变更说明（来自上游 `60e16db7`）
