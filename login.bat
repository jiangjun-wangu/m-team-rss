@echo off
REM Docker登录脚本（Windows）

echo 正在登录Docker...
docker login %1 -u %2 -p %3

if %ERRORLEVEL% NEQ 0 (
    echo 登录失败！
) else (
    echo 登录完成！
)

pause
