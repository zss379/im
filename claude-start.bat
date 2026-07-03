@echo off
chcp 65001 > nul
setlocal enabledelayedexpansion

:: ==============================================
:: DeepSeek 官方 —— V4 Flash
:: ==============================================
set API_KEY=sk-882ea98255f14472aac615fc7736b3e0
set BASE_URL=https://api.deepseek.com/anthropic
set MODEL=deepseek-v4-flash

:: ==============================================
:: 配置 Claude Code 环境变量
:: ==============================================
echo.
echo ==============================================
echo     Claude Code → DeepSeek-V4-Flash（官方）
echo ==============================================
echo.
echo [1/3] 正在配置环境变量...
set ANTHROPIC_BASE_URL=%BASE_URL%
set ANTHROPIC_AUTH_TOKEN=%API_KEY%
set ANTHROPIC_MODEL=%MODEL%
set ANTHROPIC_API_KEY=

echo [2/3] 配置完成，即将启动 Claude Code...
echo.
echo 模型: %MODEL%
echo 接口: %BASE_URL%
echo API Key: 已加载（部分隐藏）
echo.
echo [3/3] 正在启动 Claude Code...
echo ==============================================
echo.

claude

endlocal
exit