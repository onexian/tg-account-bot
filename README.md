# Telegram 多人饭堂记账机器人

这是一个基于 Go 开发的 Telegram Bot，用于多人共享记录收支，适用于家庭/团队/饭堂记账等场景。

## ✨ 功能特点

- ✅ 记录收入和支出（使用 `/add` 命令，格式简洁，如：`/add +10 午餐补贴` 或 `/add -8 晚饭`）
- 📋 查看最近交易记录（通过 `/list` 命令查看全部记录）
- 💰 查看当前总余额（使用 `/balance` 命令快速获取当前总资产）
- 📊 查看每人收支统计（使用 `/summary` 命令）
- 📆 查看本周/上周支出总额（使用 `/week` 和 `/week last`）
- 🗓 查看本月/上月支出总额（使用 `/month` 和 `/month last`）
- 👥 支持多用户，自动识别 Telegram 用户信息

## 📦 安装要求

- Go 1.21 或更高版本
- MySQL 数据库
- 一个 Telegram Bot Token（通过 [@BotFather](https://t.me/BotFather) 创建）

## ⚙️ 配置说明

1. 复制配置文件模板：

```bash
cp example.env .env

```

2. mysql 表结构：

```mysql

CREATE TABLE `transactions`
(
    `id`         bigint                                               NOT NULL AUTO_INCREMENT,
    `user_id`    bigint                                               NOT NULL,
    `type`       enum ('income','expense') COLLATE utf8mb4_general_ci NOT NULL,
    `amount`     decimal(10, 2)                                       NOT NULL,
    `note`       text COLLATE utf8mb4_general_ci,
    `created_at` timestamp                                            NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `user_id` (`user_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci;


CREATE TABLE `users`
(
    `id`         bigint    NOT NULL,
    `username`   varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
    `first_name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
    `last_name`  varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
    `created_at` timestamp NULL                          DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci;
```