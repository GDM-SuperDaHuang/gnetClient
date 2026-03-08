@echo off
chcp 65001 >nul
setlocal

:: protoc.exe 路径
set "PROTOC_BIN=E:\Softstore\protoc-28.2-win64\bin\protoc.exe"

:: protoc-gen-go 必须在 PATH
set "PATH=%PATH%;E:\gowork\bin"

echo 当前路径: %cd%
echo.

if not exist "protobuf\proto" (
    echo ❌ protobuf\proto 不存在
    pause
    exit /b 1
)

if exist "protobuf\pbGo" (
    rmdir /s /q protobuf\pbGo
)
mkdir protobuf\pbGo

echo 🚀 生成 Go 文件...
for %%f in (protobuf\proto\*.proto) do (
    "%PROTOC_BIN%" --proto_path=protobuf\proto --go_out=protobuf/pbGo --go_opt=paths=source_relative "%%f"
    if errorlevel 1 (
        echo ❌ protoc 生成失败: %%f
        pause
        exit /b 1
    )
)

echo ✅ 完成！
dir protobuf\pbGo /s /b
pause
endlocal
