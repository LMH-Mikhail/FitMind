# FitMind

Dress with Intelligence

FitMind 是一个移动端 AI 穿搭助手。当前代码落地的是第一版电子衣橱功能：

```text
Flutter App
  |
  v
Go Backend
  |
  v
PostgreSQL
```

## 目录

```text
backend/    Go 后端，controller-service-manager 结构
mobile/     Flutter 移动端
schema.sql  PostgreSQL 表结构
DESIGN.md   第一版设计文档
```

## 建库

你的 PostgreSQL 容器映射为 `5433:5432` 时，在项目根目录执行：

```bash
psql -h localhost -p 5433 -U fitmind -d fitmind_db -f schema.sql
```

如果本机没有 `psql`，可以把文件复制进容器后执行：

```bash
docker cp schema.sql fitmind:/tmp/schema.sql
docker exec -it fitmind bash
PGPASSWORD=你的密码 psql -h localhost -U fitmind -d fitmind_db -f /tmp/schema.sql
```

## 启动后端

```bash
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/server
```

## 启动前端

```bash
cd mobile
flutter create . --platforms=ios,android
flutter pub get
flutter run
```

本机当前没有 Flutter SDK 时，先安装 Flutter，再执行上面的命令。
