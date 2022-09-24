## Сервис для подачи объявлений
![Project language][badge_language]
[![Test & Lint Status][badge_build]][link_build]
[![codecov](https://codecov.io/gh/nizhikebinesi/golang-test-task/graph/badge.svg?token=JJVKAZ8PWX)](https://codecov.io/gh/nizhikebinesi/golang-test-task)
[![Twitter Follow](https://img.shields.io/twitter/follow/nizhikebinesi)](https://twitter.com/nizhikebinesi)


[badge_build]:https://img.shields.io/github/workflow/status/nizhikebinesi/golang-test-task/Check%20on%20PRs%20and%20push
[badge_language]:https://img.shields.io/badge/language-go_1.18-blue.svg?longCache=true
[link_build]:https://github.com/nizhikebinesi/golang-test-task/actions


### Как запустить
0. Установить Docker и [Docker-Compose](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-compose-on-ubuntu-20-04-ru)
1. `docker-compose build && docker-compose up -d`

### API
Address: `http://localhost:8888`
Prefix: `/api/v0.1`

| Endpoint | Method | Description                                |
| ---   | ------------- |--------------------------------------------|
| `/create_ad` | `POST` | Создание объявления |                       
| `/get_ad` | `GET` | Получение объявления |                      
| `/list_ads` | `GET` | Получение списка объявлений(для пагинции)  |

### TODOs
1. [ x ] Добавить `Sentry`
2. [ ] Добавить `Prometheus`, `Grafana`
3. [ ] Добавить `Master-Slave репликацию` для `Postgres`
4. [ ] Добавить `HA Proxy`/`Consul`/`pgpool`/`pgbouncer`
5. [ ] Добавить `nginx`(с `Consul`) и запустить копии сервиса
6. [ ] Сгенерировать и хостить `Swagger`-документацию
7. [ ] Добавить `DELETE`(удаления записей) и `PUT`(изменения записей) методы в сервис
8. [ ] Добавить тестирование через `dockertest`

### **Задача**

Разработать сервис для подачи объявлений с сохранением в базе данных. 
Сервис должен предоставлять API, работающее поверх HTTP в формате JSON.

### **Требования**

- Язык программирования — Go;
- Готовую версию выложить на Github;
- Простая инструкция для запуска(в идеале 
— с возможностью запустить через `docker-compose up`, но это необязательно);
- 3 метода:
    - получение списка объявлений,
    - получение одного объявления,
    - создание объявления;
- Валидация полей:
    - не больше 3 ссылок на фото,
    - описание не больше 1000 символов,
    - название не больше 200 символов;

Если есть сомнения по деталям — решение принять самостоятельно, 
но в своём `README.md` рекомендуем выписать вопросы и принятые решения по ним.

### Ограничения по времени

2-4 часа на выполнение. Если что-то не укладывается в указанное время, 
то реализовать задачу по степени важности функционала. 
Мы не требуем выполнить абсолютно всё. Здесь важны умение приоритизировать и 
чистота кода.

### **Детали**

**Метод получения списка объявлений**

- Пагинация: на одной странице должно присутствовать 10 объявлений;
- Cортировки: по цене(возрастание/убывание) и 
по дате создания(возрастание/убывание);
- Поля в ответе: название объявления, 
ссылка на главное фото (первое в списке), цена.

**Метод получения конкретного объявления**

- Обязательные поля в ответе: название объявления, цена, ссылка на главное фото;
- Опциональные поля (можно запросить, передав параметр fields): 
описание, ссылки на все фото.

**Метод создания объявления:**

- Принимает все вышеперечисленные поля: название, описание, 
несколько ссылок на фотографии
(сами фото загружать никуда не требуется), цена;
- Возвращает ID созданного объявления и код результата (ошибка или успех).
