# Управление постами

## Обзор

PostService управляет созданием, редактированием, публикацией постов, а также лайками и закладками. Посты - это основной пользовательский контент платформы.

## gRPC Service: PostService

Proto файл: `proto/post.proto`

## Сущность

### Post

```protobuf
message Post {
  string id
  string author_id
  string author_name
  string community_id
  string community_name
  string title
  google.protobuf.Struct content      // Структурированный JSON контент
  PostStatus status                    // draft or published
  int32 like_count
  int32 comment_count
  bool is_liked_by_me
  bool is_bookmarked_by_me
  google.protobuf.Timestamp published_at
  google.protobuf.Timestamp created_at
  google.protobuf.Timestamp updated_at
}
```

### PostStatus

```protobuf
enum PostStatus {
  POST_STATUS_UNSPECIFIED = 0
  POST_STATUS_DRAFT       = 1
  POST_STATUS_PUBLISHED   = 2
}
```

## Endpoints

### Create

**RPC:** `Create(CreateRequest) returns (CreateResponse)`  
**HTTP:** `POST /posts`  
**FR:** FR-017-020, FR-327-334

Создание нового поста.

**Request:**

```protobuf
message CreateRequest {
  string community_id
  string title                    // 3-300 символов
  google.protobuf.Struct content  // JSON
  PostStatus status               // draft or published
}
```

**Response:**

```protobuf
message CreateResponse {
  Post post
}
```

**Требования:**

- Пользователь должен быть верифицирован (FR-009, FR-329)
- Пользователь должен быть участником сообщества или сообщество разрешает постинг неучастников (FR-330)
- Title 3-300 символов (FR-331, FR-368)
- Content должен быть валидным JSON (FR-332)
- Status: draft или published
- Автор устанавливается в текущего пользователя (FR-333)
- Посты принадлежат ровно одному сообществу (FR-019, FR-020)
- Возврат полного объекта Post (FR-334)

**Ошибки:**

- Пользователь не верифицирован
- Не участник сообщества
- Title невалидной длины
- Content не является валидным JSON
- Сообщество забанено
- Сообщество не найдено

---

### Get

**RPC:** `Get(GetRequest) returns (GetResponse)`  
**HTTP:** `GET /posts/{post_id}`  
**FR:** FR-208, FR-213

Получение поста по ID.

**Request:**

```protobuf
message GetRequest {
  string post_id
}
```

**Response:**

```protobuf
message GetResponse {
  Post post
}
```

**Требования:**

- Возврат полной информации (FR-213):
  - title, content
  - author info (id, username)
  - community info (id, name)
  - status
  - like_count, comment_count
  - timestamps (created_at, updated_at, published_at)
- Поля is_liked_by_me и is_bookmarked_by_me требуют аутентификации
- Черновики видны только автору

**Ошибки:**

- Пост не найден
- Попытка доступа к чужому черновику

---

### Update

**RPC:** `Update(UpdateRequest) returns (UpdateResponse)`  
**HTTP:** `PATCH /posts/{post_id}`  
**FR:** FR-023, FR-376-383

Обновление существующего поста.

**Request:**

```protobuf
message UpdateRequest {
  string post_id
  optional string title
  optional google.protobuf.Struct content
}
```

**Response:**

```protobuf
message UpdateResponse {
  Post post
}
```

**Требования:**

- Требуется быть автором или иметь edit_any_post permission (FR-132, FR-378)
- Все поля опциональны
- Title 3-300 символов если указан (FR-379)
- Content должен быть валидным JSON если указан (FR-380)
- НЕ может изменять (FR-381):
  - author
  - community_id
  - created_at
- Автоматическое обновление updated_at (FR-382)
- Возврат обновленного Post (FR-383)

**Ошибки:**

- Недостаточно прав
- Title невалидной длины
- Content не является валидным JSON
- Пост не найден

---

### Delete

**RPC:** `Delete(DeleteRequest) returns (DeleteResponse)`  
**HTTP:** `DELETE /posts/{post_id}`  
**FR:** FR-209

Удаление поста.

**Request:**

```protobuf
message DeleteRequest {
  string post_id
}
```

**Response:**

```protobuf
message DeleteResponse {
  string message
}
```

**Требования:**

- Автор может удалить с delete_own_post permission
- Модератор может удалить с delete_any_post permission (FR-127)
- Cascade удаление:
  - Все комментарии к посту
  - Все лайки
  - Все закладки

**Ошибки:**

- Недостаточно прав
- Пост не найден

---

### ListComments

**RPC:** `ListComments(ListCommentsRequest) returns (ListCommentsResponse)`  
**HTTP:** `GET /posts/{post_id}/comments`  
**FR:** FR-217, FR-220, FR-221

Получение списка комментариев к посту.

**Request:**

```protobuf
message ListCommentsRequest {
  string post_id
  string cursor
  int32 limit
}
```

**Response:**

```protobuf
message ListCommentsResponse {
  repeated Comment comments
  string next_cursor
  bool has_more
}
```

**Требования:**

- Cursor-based пагинация
- Сортировка по дате создания по возрастанию (старые первые, хронологическое чтение) (FR-220)
- Поддержка вложенных комментариев через parent_comment_id (FR-221)

---

### Publish

**RPC:** `Publish(PublishRequest) returns (PublishResponse)`  
**HTTP:** `POST /posts/{post_id}/publish`  
**FR:** FR-210

Публикация черновика.

**Request:**

```protobuf
message PublishRequest {
  string post_id
}
```

**Response:**

```protobuf
message PublishResponse {
  Post post
}
```

**Требования:**

- Изменение статуса draft → published
- Установка published_at timestamp
- Пост становится виден в лентах
- Отправка уведомлений подписчикам и участникам сообщества

**Ошибки:**

- Недостаточно прав (только автор)
- Пост уже опубликован
- Пост не найден

---

### Unpublish

**RPC:** `Unpublish(UnpublishRequest) returns (UnpublishResponse)`  
**HTTP:** `POST /posts/{post_id}/unpublish`  
**FR:** FR-211

Снятие поста с публикации (возврат в черновики).

**Request:**

```protobuf
message UnpublishRequest {
  string post_id
}
```

**Response:**

```protobuf
message UnpublishResponse {
  Post post
}
```

**Требования:**

- Изменение статуса published → draft
- Требуется unpublish_post permission (FR-039, FR-127)
- Пост исчезает из публичных лент
- Остается доступен автору

**Ошибки:**

- Недостаточно прав
- Пост уже в черновиках
- Пост не найден

---

## Лайки

### Like

**RPC:** `Like(LikeRequest) returns (LikeResponse)`  
**HTTP:** `POST /posts/{post_id}/like`  
**FR:** FR-021, FR-171, FR-175, FR-177, FR-178

Лайк поста.

**Request:**

```protobuf
message LikeRequest {
  string post_id
}
```

**Response:**

```protobuf
message LikeResponse {
  string message
  int32 new_like_count
}
```

**Требования:**

- Пользователь должен быть верифицирован
- Идемпотентность: повторный лайк возвращает success (FR-175)
- Немедленное обновление like_count (FR-177)
- Обновление репутации автора (FR-178, FR-053)
- Создание уведомления автору

**Ошибки:**

- Пользователь не верифицирован
- Пользователь забанен (FR-063)
- Пост не найден

---

### Unlike

**RPC:** `Unlike(UnlikeRequest) returns (UnlikeResponse)`  
**HTTP:** `DELETE /posts/{post_id}/like`  
**FR:** FR-172, FR-176

Удаление лайка с поста.

**Request:**

```protobuf
message UnlikeRequest {
  string post_id
}
```

**Response:**

```protobuf
message UnlikeResponse {
  string message
  int32 new_like_count
}
```

**Требования:**

- Идемпотентность: удаление несуществующего лайка возвращает success (FR-176)
- Немедленное обновление like_count (FR-177)
- Обновление репутации автора (FR-178)

---

## Закладки

### CreateBookmark

**RPC:** `CreateBookmark(CreateBookmarkRequest) returns (CreateBookmarkResponse)`  
**HTTP:** `POST /posts/{post_id}/bookmark`  
**FR:** FR-022, FR-165, FR-169

Добавление поста в закладки.

**Request:**

```protobuf
message CreateBookmarkRequest {
  string post_id
}
```

**Response:**

```protobuf
message CreateBookmarkResponse {
  string message
}
```

**Требования:**

- Пользователь должен быть верифицирован
- Идемпотентность: повторное добавление возвращает success (FR-169)
- Закладки приватны (видны только владельцу) (FR-170)

**Ошибки:**

- Пользователь не верифицирован
- Пользователь забанен (FR-063)
- Пост не найден

---

### DeleteBookmark

**RPC:** `DeleteBookmark(DeleteBookmarkRequest) returns (DeleteBookmarkResponse)`  
**HTTP:** `DELETE /posts/{post_id}/bookmark`  
**FR:** FR-166

Удаление поста из закладок.

**Request:**

```protobuf
message DeleteBookmarkRequest {
  string post_id
}
```

**Response:**

```protobuf
message DeleteBookmarkResponse {
  string message
}
```

**Требования:**

- Идемпотентность: удаление несуществующей закладки возвращает success

---

### ListBookmarks

**RPC:** `ListBookmarks(ListBookmarksRequest) returns (ListBookmarksResponse)`  
**HTTP:** `GET /posts/bookmarks`  
**FR:** FR-167, FR-168, FR-170

Получение списка закладок пользователя.

**Request:**

```protobuf
message ListBookmarksRequest {
  string cursor
  int32 limit
}
```

**Response:**

```protobuf
message ListBookmarksResponse {
  repeated Post posts
  string next_cursor
  bool has_more
}
```

**Требования:**

- Cursor-based пагинация
- Сортировка по дате добавления в закладки (новые первые) (FR-168)
- Только собственные закладки (FR-170)
- Требуется аутентификация

---

## Статусы постов

### Draft (Черновик)

- Создается со статусом draft если не указано published
- Видим только автору
- Не появляется в публичных лентах
- Можно редактировать без ограничений
- Можно опубликовать через Publish

### Published (Опубликован)

- Виден всем пользователям
- Появляется в лентах сообщества
- Появляется в персонализированных лентах подписчиков
- Может быть отредактирован
- Может быть снят с публикации модератором (Unpublish)
- Участвует в алгоритме ранжирования

## JSON Content

### Формат

Content хранится как `google.protobuf.Struct` - произвольный JSON. В ORM модели реализовано как `json.RawMessage` с типом `JSONB` в PostgreSQL. gRPC сервер корректно преобразует `google.protobuf.Struct` в `json.RawMessage` при сохранении и обратно при возврате данных.

### Рекомендуемая структура

Зависит от rich text editor на клиенте. Примеры форматов:

- Draft.js
- ProseMirror
- Quill Delta
- Custom JSON

### Валидация

- Должен быть валидным JSON
- Размер не жестко ограничен (рекомендуется до 1MB)
- Бинарные данные НЕ хранятся в content
- Медиа хранится в S3, в content только ссылки

## Счетчики

### like_count

- Количество уникальных пользователей, поставивших лайк
- Обновляется при Like/Unlike
- Используется в алгоритме ранжирования

### comment_count

- Количество всех комментариев к посту (включая вложенные)
- Обновляется при создании/удалении комментария
- Используется в алгоритме ранжирования

## Репутация автора

При получении/удалении лайка на посте:

- Обновляется reputation автора (FR-053, FR-178)
- post_likes в статистике пользователя
- total_likes_received в статистике пользователя

## Ограничения

### Создание

- Только верифицированные пользователи (FR-009)
- Должны быть участниками сообщества (по умолчанию)
- Без лимита на количество постов (FR-017)

### Редактирование

- Автор всегда может редактировать свои посты
- Модераторы с edit_any_post могут редактировать любые

### Удаление

- Автор с delete_own_post
- Модератор с delete_any_post

### Забаненные пользователи

НЕ могут (FR-061, FR-063):

- Создавать новые посты
- Редактировать существующие
- Лайкать посты
- Добавлять в закладки

Их существующие посты:

- Остаются видимыми (FR-060)
- Отображаются с индикатором "banned user" (FR-062)

## Каскадное удаление

При удалении поста удаляются:

- Все комментарии (с их лайками)
- Все лайки поста
- Все закладки поста
- Связи в уведомлениях (soft delete)

При удалении сообщества удаляются:

- Все посты сообщества (с каскадом выше)

## Производительность

### Индексы

Рекомендуемые индексы:

- (community_id, published_at) для лент сообщества
- (author_id, created_at) для постов пользователя
- (status, author_id) для черновиков
- Full-text на title для поиска

### Кеширование

Рекомендуется кешировать:

- Счетчики (like_count, comment_count)
- Hot посты в лентах
- Данные автора и сообщества
