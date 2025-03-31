@echo off
REM Build the server executable
go build -o .\tmp\main.exe ./cmd/server/main.go

REM Build the WebAssembly file
set GOOS=js
set GOARCH=wasm
go build -o .\webassembly\json.wasm ./cmd/wasm/main.go
set GOOS=
set GOARCH=