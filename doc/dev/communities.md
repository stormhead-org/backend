# Управление сообществами

## Обзор

CommunityService управляет созданием, настройками, членством и модерацией сообществ. Сообщество - это организационная единица для группировки постов с собственным владельцем, модераторами и правилами.

## gRPC Service: CommunityService

Proto файл: `proto/community.proto`

## Сущность

### Community

```protobuf
message Community {
  string id
  string owner_id
  string owner_name
  string name
  string description
  string rules
  int32 member_count
  int32 post_count
  int32 reputation
  bool is_banned
  google.protobuf.Timestamp created_at
  google.protobuf.Timestamp updated_at
}
```

## Endpoints

### Create

**RPC:** `Create(CreateRequest) returns (CreateResponse)`  
**HTTP:** `POST /communities`  
**FR:** FR-010, FR-345-354

Создание нового сообщества.

**Request:**

```protobuf
message CreateRequest {
  string name         // 3-50 символов, уникальное
  string description  // max 500 символов
  string rules
}
```

**Response:**

```protobuf
message CreateResponse {
  Community community
}
```

**Требования:**

- Пользователь должен быть верифицирован (FR-009, FR-347)
- Имя 3-50 символов (FR-349)
- Имя должно быть уникальным (FR-348)
- Description максимум 500 символов (FR-350)
- Создатель автоматически становится владельцем (FR-011, FR-351)
- Автоматическое создание роли сообщества @everyone (FR-096, FR-352)
- Создатель автоматически получает роль @everyone сообщества (FR-098, FR-353)
- Возврат полного объекта Community (FR-354)

**Ошибки:**

- Пользователь не верифицирован
- Имя уже занято
- Имя короче 3 или длиннее 50 символов
- Description длиннее 500 символов

---

### Get

**RPC:** `Get(GetRequest) returns (GetResponse)`  
**HTTP:** `GET /communities/{community_id}`  
**FR:** FR-224, FR-230

Получение информации о сообществе.

**Request:**

```protobuf
message GetRequest {
  string community_id
}
```

**Response:**

```protobuf
message GetResponse {
  Community community
}
```

**Требования:**

- Возврат полной информации (FR-230):
  - name, description, rules
  - owner info (id, username)
  - member_count
  - post_count
  - created_at
  - banned status

---

### Update

**RPC:** `Update(UpdateRequest) returns (UpdateResponse)`  
**HTTP:** `PATCH /communities/{community_id}`  
**FR:** FR-012, FR-391-399

Обновление настроек сообщества.

**Request:**

```protobuf
message UpdateRequest {
  string community_id
  optional string name
  optional string description
  optional string rules
}
```

**Response:**

```protobuf
message UpdateResponse {
  Community community
}
```

**Требования:**

- Требуется быть владельцем или иметь edit_community_settings permission (FR-129, FR-393)
- Все поля опциональны
- Имя 3-50 символов если указано (FR-395)
- Имя должно быть уникальным если указано (FR-394)
- Description максимум 500 символов если указано (FR-396)
- НЕ может изменять (FR-397):
  - owner
  - created_at
  - member_count
- Автоматическое обновление updated_at (FR-398)
- Возврат обновленного Community (FR-399)

**Ошибки:**

- Недостаточно прав
- Имя уже занято
- Невалидная длина имени или description

---

### Delete

**RPC:** `Delete(DeleteRequest) returns (DeleteResponse)`  
**HTTP:** `DELETE /communities/{community_id}`  
**FR:** FR-225, FR-236, FR-237

Удаление сообщества.

**Request:**

```protobuf
message DeleteRequest {
  string community_id
  bool confirm  // должно быть true
}
```

**Response:**

```protobuf
message DeleteResponse {
  string message
}
```

**Требования:**

- Требуется delete_community permission или быть владельцем (FR-225)
- Обязательное подтверждение через confirm=true (FR-237)
- Cascade удаление (FR-236):
  - Все посты сообщества
  - Все комментарии к постам
  - Все роли сообщества
- Операция необратима (FR-237)

**Ошибки:**

- Недостаточно прав
- confirm != true
- Сообщество не найдено

---

### ListCommunities

**RPC:** `ListCommunities(ListCommunitiesRequest) returns (ListCommunitiesResponse)`  
**HTTP:** `GET /communities`  
**FR:** FR-226, FR-231

Получение списка всех сообществ платформы.

**Request:**

```protobuf
message ListCommunitiesRequest {
  string cursor
  int32 limit
}
```

**Response:**

```protobuf
message ListCommunitiesResponse {
  repeated Community communities
  string next_cursor
  bool has_more
}
```

**Требования:**

- Cursor-based пагинация
- Сортировка по member_count в обратном порядке (самые популярные первые) (FR-231)
- Скрытие забаненных сообществ от обычных пользователей (FR-233)
- Забаненные видны только модераторам

---

### Join

**RPC:** `Join(JoinRequest) returns (JoinResponse)`  
**HTTP:** `POST /communities/{community_id}/join`  
**FR:** FR-016, FR-179, FR-181, FR-183

Вступление в сообщество.

**Request:**

```protobuf
message JoinRequest {
  string community_id
}
```

**Response:**

```protobuf
message JoinResponse {
  string message
}
```

**Требования:**

- Пользователь должен быть верифицирован (FR-009)
- Автоматическое назначение роли @everyone сообщества (FR-099, FR-181)
- Идемпотентность: повторное вступление возвращает success (FR-183)
- Увеличение member_count

**Ошибки:**

- Пользователь не верифицирован
- Сообщество забанено
- Сообщество не найдено

---

### Leave

**RPC:** `Leave(LeaveRequest) returns (LeaveResponse)`  
**HTTP:** `POST /communities/{community_id}/leave`  
**FR:** FR-016, FR-180, FR-182, FR-184, FR-185, FR-452-454

Выход из сообщества.

**Request:**

```protobuf
message LeaveRequest {
  string community_id
}
```

**Response:**

```protobuf
message LeaveResponse {
  string message
}
```

**Требования:**

- Автоматическое удаление роли @everyone сообщества (FR-104, FR-182)
- Платформенная роль @everyone остается (FR-105)
- Идемпотентность: выход из несостоящего сообщества возвращает success (FR-184)
- Владелец НЕ может выйти из собственного сообщества (FR-185, FR-452-454)
- Владелец всегда считается участником сообщества (FR-452)
- Валидация что пользователь не является владельцем перед выходом (FR-454)
- Уменьшение member_count

**Ошибки:**

- Владелец пытается выйти (требуется сначала передать владение другому участнику сообщества)

---

### Ban

**RPC:** `Ban(BanRequest) returns (BanResponse)`  
**HTTP:** `POST /communities/{community_id}/ban`  
**FR:** FR-228, FR-233-235

Бан сообщества на платформе.

**Request:**

```protobuf
message BanRequest {
  string community_id
  string reason
}
```

**Response:**

```protobuf
message BanResponse {
  string message
}
```

**Требования:**

- Требуется platform moderation permission (FR-228)
- Сообщество скрывается из ListCommunities для обычных пользователей (FR-233)
- Участники НЕ могут создавать новые посты (FR-234)
- Существующие посты остаются видимыми с индикатором "banned community" (FR-235)
- Логирование действия модерации (FR-267)

**Ошибки:**

- Недостаточно прав
- Сообщество не найдено
- Сообщество уже забанено

---

### Unban

**RPC:** `Unban(UnbanRequest) returns (UnbanResponse)`  
**HTTP:** `POST /communities/{community_id}/unban`  
**FR:** FR-229

Снятие бана с сообщества.

**Request:**

```protobuf
message UnbanRequest {
  string community_id
}
```

**Response:**

```protobuf
message UnbanResponse {
  string message
}
```

**Требования:**

- Требуется platform moderation permission (FR-229)
- Восстановление возможности создавать посты
- Сообщество снова появляется в ListCommunities
- Логирование действия модерации (FR-267)

**Ошибки:**

- Недостаточно прав
- Сообщество не найдено
- Сообщество не забанено

---

### TransferOwnership

**RPC:** `TransferOwnership(TransferOwnershipRequest) returns (TransferOwnershipResponse)`  
**HTTP:** `POST /communities/{community_id}/transfer-ownership`  
**FR:** FR-014, FR-015

Передача владения сообществом.

**Request:**

```protobuf
message TransferOwnershipRequest {
  string community_id
  string new_owner_id
}
```

**Response:**

```protobuf
message TransferOwnershipResponse {
  string message
}
```

**Требования:**

- Требуется быть текущим владельцем
- Новый владелец должен быть участником сообщества
- Новый владелец получает все привилегии (FR-015)
- Предыдущий владелец становится обычным участником (FR-015)
- Атомарная операция

**Ошибки:**

- Только владелец может передать владение
- Новый владелец не является участником
- Новый владелец не найден

---

## Членство в сообществе

### Автоматические роли

При создании сообщества:

- Создается роль @everyone для сообщества (FR-096)
- Создатель получает роль @everyone (FR-098)
- Создатель становится owner

При вступлении:

- Пользователь автоматически получает роль @everyone сообщества (FR-099)

При выходе:

- Роль @everyone сообщества автоматически удаляется (FR-104)
- Платформенная роль @everyone сохраняется

### Разрешения по умолчанию

Роль @everyone сообщества по умолчанию имеет (FR-110):

- report_content: возможность жаловаться на контент в сообществе

Владелец может редактировать разрешения роли @everyone (FR-103).

### Иерархия владения

1. **Владелец сообщества (owner)**

   - Полный контроль над сообществом
   - Всегда считается участником сообщества (FR-452)
   - Может назначать модераторов
   - Может редактировать настройки
   - Может передать владение другому участнику
   - НЕ может выйти из сообщества до передачи владения (FR-453)

2. **Модераторы**

   - Получают специальные роли с разрешениями
   - Могут управлять контентом в рамках разрешений
   - Назначаются владельцем через систему ролей

3. **Участники**
   - Могут создавать посты
   - Могут комментировать
   - Имеют роль @everyone сообщества

## Модерация сообществ

### Бан сообщества

Когда сообщество забанено:

- Скрывается из публичных списков (FR-233)
- Новые посты запрещены (FR-234)
- Существующие посты видимы с меткой (FR-235)
- Модераторы все еще видят в списках
- Участники сохраняют членство

### Разблокировка

При снятии бана:

- Возвращается в публичные списки
- Восстанавливается возможность постить
- Метка "banned" убирается с постов

## Статистика сообщества

### Счетчики

- **member_count**: количество участников
- **post_count**: количество постов
- **reputation**: рейтинг на основе вовлеченности (FR-455-458)

### Обновление

Счетчики обновляются автоматически:

- member_count при join/leave
- post_count при создании/удалении постов
- reputation при лайках постов и создании комментариев

## Cascade удаление

При удалении сообщества каскадно удаляются (FR-236):

1. **Все посты**

   - Включая их лайки
   - Включая их закладки

2. **Все комментарии**

   - К постам сообщества
   - Включая лайки комментариев

3. **Все роли**

   - Включая @everyone
   - Роли автоматически убираются у всех пользователей

4. **Членство**
   - Все связи membership удаляются

## Валидация

### Имя сообщества

- Минимум: 3 символа
- Максимум: 50 символов
- Должно быть уникальным на платформе
- Рекомендуется использовать буквы, цифры, дефисы

### Description

- Максимум: 500 символов
- Может содержать Markdown (зависит от клиента)

### Rules

- Без жестких ограничений длины
- Рекомендуется структурированный формат
- Может содержать Markdown

---

## Рейтинг сообщества

### Расчет (FR-455-458)

Рейтинг сообщества рассчитывается по формуле:

```
Reputation = sum(post_likes) + (sum(comments) × 0.1)
```

Где:

- **sum(post_likes)** - сумма всех лайков на постах сообщества
- **sum(comments)** - общее количество комментариев к постам сообщества
- **0.1** - весовой коэффициент для комментариев (10 комментариев = 1 балл)

### Компоненты

**Лайки постов:**

- Каждый лайк на любом посте сообщества добавляет 1 балл
- Учитываются только опубликованные посты
- Лайки на черновиках не учитываются

**Комментарии:**

- Каждый комментарий добавляет 0.1 балла
- Учитываются все комментарии (включая вложенные)
- Удаленные комментарии не учитываются

### Обновление

Reputation обновляется автоматически (FR-457):

- При получении/удалении лайка на посте
- При создании нового комментария
- При удалении комментария

### Отображение

Reputation виден (FR-458):

- В профиле сообщества (GetCommunity)
- В списках сообществ (ListCommunities)
- В карточках сообществ

### Использование

**Сортировка:**

- Сообщества могут сортироваться по reputation
- Альтернатива сортировке по member_count

**Индикатор качества:**

- Высокий reputation = активное сообщество с качественным контентом
- Показывает вовлеченность участников

### Реализация

**Расчет:**

- Может выполняться на лету при запросе
- Или: кешироваться и обновляться триггерами
- Рекомендуется материализованное представление для производительности

**Индексы:**

- (community_id, reputation) для сортировки
- Обновление через триггеры при изменении лайков/комментариев

---

## Ограничения

### Создание

- Только верифицированные пользователи (FR-009)
- Без лимита на количество создаваемых сообществ (FR-010)

### Владение

- Пользователь может владеть неограниченным количеством сообществ
- Требуется передать владение перед удалением аккаунта

### Членство

- Пользователь может быть участником неограниченного количества сообществ
- Отслеживается в joined_communities_count профиля
