---
title: Bot @Trigger 架构设计
status: proposal
version: 0.1
created: 2026-07-03
---

# Bot @Trigger 架构设计

## 摘要

实现用户在 IM 中 @机器人后，消息经异步通道触发外部系统 Webhook，外部系统响应后自动回复到会话的能力。同步支持 sync/async/SSE 三种响应模式，并为 AI 流式对话预留 SSE 接入点。

## 范围

| 包含 | 不包含 |
|------|--------|
| @触发的完整消息链路 | 系统机器人的内部触发逻辑 |
| Webhook 三种响应模式 | 机器人管理的 UI/UX |
| SSE 流式接入预留 | AI 模型服务的对接 |
| 机器人身份与 OpenIM 集成 | |

## 架构决策记录

| ADR | 决策 | 理由 |
|-----|------|------|
| ADR-01 | 机器人 = OpenIM 用户 | @mention 天然支持，共享权限体系 |
| ADR-02 | 触发点 = message-svc → Kafka bot_trigger | 非阻塞，不耦合主消息路径 |
| ADR-03 | Webhook 三模式可配 (sync/async/sse) | 覆盖从简单指令到 AI 对话的全部场景 |
| ADR-04 | 机器人回复走标准消息通道 | 经敏感词/频控/权限校验 |
| ADR-05 | SSE 预留 bot-svc 连接池 | 为 AI 流式输出做好准备 |
