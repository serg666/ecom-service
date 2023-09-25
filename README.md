# ECOM service

## Миграции

Для запуска миграций установите **Migrate CLI**

```bash
https://github.com/golang-migrate/migrate/tree/master/cmd/migrate
```

Так же можно использовать команду
```bash
make migrate
```

Более подробная информация по установке [тут](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#installation)

Для взаимодействия с БД, используйте пользователя **ecom-service** (в миграциях)

```postgres
CREATE DATABASE ecom-service;

CREATE USER ecom-service WITH ENCRYPTED PASSWORD 'qazwsx';
GRANT ALL PRIVILEGES ON DATABASE ecom-service TO ecom-service;
```

## Генерация документции swagger

Для генерации документации установите **Swag**

```bash
https://github.com/swaggo/swag/
```

Так же можно использовать команду
```bash
make swag
```

Более подробная информация по использованию [тут](https://github.com/swaggo/swag/#getting-started)

Сгенерируйте документацию командой
```bash
make swag-init
```

## Сборка проекта

Для сборки проекта выполните команду
```bash
make all
```

Для запуска тестов выполните команду
```bash
make test
```

## Конфигурация и запуск

Перед запуском проекта отредактируйте файл config/config.yml

Перед первоначальным запуском выполните команду
```bash
migrate -path migrations/ -database $DB_DSN up 1
```

Затем выполните миграции, запустив команду
```bash
migrate -path migrations/ -database $DB_DSN up
```

Подробная информация по использованию команды [тут](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#usage)

Перед первоначальным запуском приложения выполните загрузку списка валют, выполнив команду
```bash
/build/bin/loadcurrencies
```

Инструкция по использованию команды доступна при указании ключа -help
```bash
./build/bin/loadcurrencies --help
Usage of ./build/bin/loadcurrencies:
  -config string
        path to config file (default "./config.yml")
  -currencies string
        path to currencies file (default "./currencies.xml")
```

Для запуска приложения выполните команду
```bash
make run
```

## Команды, запускаемые из крона

Для загрузки курсов валют добавьте в крон выполнение следующей команды
```bash
./build/bin/loadrates
```

Инструкция по использованию команды доступна при указании ключа --help
```bash
./build/bin/loadrates --help
Usage of ./build/bin/loadrates:
  -config string
        path to config file (default "./config.yml")
```
