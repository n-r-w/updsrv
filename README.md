# updsrv
Сервер для хранения и выдачи обновления ПО по запросу. Вычисляет разницу между версиями и присылает только измененные файлы. Результаты вычисления кэшируются в БД для исключения высокой нагрузки на CPU

## Поддерживает следующие операции

Добавить обновление

    curl --location --request POST 'http://localhost:8081/api/add' \
    --header 'X-Authorization: dbda0fba4da680c615340d6faa2868eb5413c3b837640078b87149872257f842' \
    --form 'update=@"/home/we/product21.zip"' \
    --form 'buildTime="2022-06-17T07:30"' \
    --form 'channel="HRFILE_PROD1"' \
    --form 'version="4.1.2.9"' \
    --form 'info="информация"' \
    --form 'enabled="true"'
    
Проверить наличие обновлений

    curl --location --request POST 'http://localhost:8081/api/check' \
    --header 'X-Authorization: dbda0fba4da680c615340d6faa2868eb5413c3b837640078b87149872257f842' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "channel": "HRFILE_PROD",
        "version": {
            "major": 4,
            "minor": 1,
            "patch": 1,
            "revision": 8
        }
    }'

Получить обновление

    curl --location --request POST 'http://localhost:8081/api/update' \
    --header 'X-Authorization: dbda0fba4da680c615340d6faa2868eb5413c3b837640078b87149872257f842' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "channel": "HRFILE_PROD1",
        "version": {
            "major": 4,
            "minor": 1,
            "patch": 1,
            "revision": 8
        }
    }'
