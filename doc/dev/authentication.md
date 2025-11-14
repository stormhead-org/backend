# Система аутентификации и авторизации

## Обзор

Система использует JWT токены с двухуровневой схемой: access token (короткоживущий) и refresh token (долгоживущий).

## gRPC Service: AuthorizationService

Proto файл: `proto/authorization.proto`

## Сущности

### User

```protobuf
message User {
  string id
  string name
  string slug
  string email
  string avatar_url
  string banner_url
  string description
  int32 reputation
  bool is_verified
  bool is_banned
  google.protobuf.Timestamp created_at
  google.protobuf.Timestamp updated_at
}
```

### Session

```protobuf
message Session {
  string session_id
  string device_info
  string ip_address
  google.protobuf.Timestamp last_activity
  google.protobuf.Timestamp created_at
}
```

## Endpoints

### Register

**RPC:** `Register(RegisterRequest) returns (RegisterResponse)`  
**HTTP:** `POST /auth/register`  
**FR:** FR-001, FR-280, FR-291, FR-292

Регистрация нового пользователя на платформе.

**Request:**

```protobuf
message RegisterRequest {
  string username  // уникальное имя пользователя
  string email     // email для верификации
  string password  // минимум 12 символов
}
```

**Response:**

```protobuf
message RegisterResponse {
  string user_id
  string message  // "Verification email sent"
}
```

**Требования:**

- Username должен быть уникальным
- Email должен быть уникальным и валидным
- Пароль минимум 12 символов (FR-070)
- Пароль проверяется через Have I Been Pwned API (FR-071)
- Скомпрометированные пароли отклоняются с ясным сообщением об ошибке (FR-072)
- Автоматическая отправка email для верификации (FR-002, FR-292)
- Автоматическое назначение платформенной роли @everyone (FR-094)
- Первый зарегистрированный пользователь становится владельцем платформы (FR-095)

**Ошибки:**

- Username уже занят
- Email уже зарегистрирован
- Пароль не соответствует требованиям
- Пароль найден в базе скомпрометированных паролей

---

### Login

**RPC:** `Login(LoginRequest) returns (LoginResponse)`  
**HTTP:** `POST /auth/login`  
**FR:** FR-004, FR-281, FR-293, FR-294

Аутентификация пользователя и получение JWT токенов.

**Request:**

```protobuf
message LoginRequest {
  string email
  string password
}
```

**Response:**

```protobuf
message LoginResponse {
  User user
  string access_token   // JWT, 15 минут
  string refresh_token  // JWT, 7 дней
}
```

**Требования:**

- Rate limiting: 5 попыток за 10 минут на IP адрес (FR-055, FR-293)
- Проверка верификации email перед разрешением входа (FR-294)
- Access token действителен 15 минут (FR-057)
- Refresh token действителен 7 дней (FR-058)
- Создание новой сессии с информацией об устройстве и IP

**Ошибки:**

- Неверный email или пароль
- Email не верифицирован
- Rate limit превышен (HTTP 429) (FR-059)
- Пользователь забанен

---

### Logout

**RPC:** `Logout(LogoutRequest) returns (LogoutResponse)`  
**HTTP:** `POST /auth/logout`  
**FR:** FR-005, FR-282, FR-302

Завершение текущей сессии пользователя.

**Request:**

```protobuf
message LogoutRequest {}
```

**Response:**

```protobuf
message LogoutResponse {
  string message
}
```

**Требования:**

- Аннулирование текущей сессии (FR-302)
- Маркировка refresh token как недействительного
- Требуется аутентификация

---

### RefreshToken

**RPC:** `RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse)`  
**HTTP:** `POST /auth/refresh`  
**FR:** FR-283, FR-295, FR-296

Обновление access token используя действующий refresh token.

**Request:**

```protobuf
message RefreshTokenRequest {
  string refresh_token
}
```

**Response:**

```protobuf
message RefreshTokenResponse {
  string access_token   // новый JWT, 15 минут
  string refresh_token  // новый JWT, 7 дней
}
```

**Требования:**

- Валидация срока действия refresh token (7 дней) (FR-295)
- Генерация нового access token с 15-минутным сроком (FR-296, FR-057)
- Генерация нового refresh token
- Старый refresh token аннулируется

**Ошибки:**

- Refresh token недействителен
- Refresh token истек
- Refresh token был отозван

---

### VerifyEmail

**RPC:** `VerifyEmail(VerifyEmailRequest) returns (VerifyEmailResponse)`  
**HTTP:** `POST /auth/verify-email`  
**FR:** FR-003, FR-284

Подтверждение email адреса через токен из письма.

**Request:**

```protobuf
message VerifyEmailRequest {
  string token  // токен из ссылки в email
}
```

**Response:**

```protobuf
message VerifyEmailResponse {
  string message
  User user
}
```

**Требования:**

- Валидация токена верификации
- Маркировка email как верифицированного
- Разблокировка возможности создавать контент (FR-009)

**Ошибки:**

- Токен недействителен
- Токен истек
- Email уже верифицирован

---

### RequestPasswordReset

**RPC:** `RequestPasswordReset(RequestPasswordResetRequest) returns (RequestPasswordResetResponse)`  
**HTTP:** `POST /auth/password-reset/request`  
**FR:** FR-006, FR-285

Инициация процесса восстановления пароля.

**Request:**

```protobuf
message RequestPasswordResetRequest {
  string email
}
```

**Response:**

```protobuf
message RequestPasswordResetResponse {
  string message  // "Password reset email sent"
}
```

**Требования:**

- Отправка email с secure токеном для сброса пароля (FR-007)
- Токен должен иметь ограниченный срок действия (рекомендуется 1 час)
- Всегда возвращать success для предотвращения enumeration атак
- Email доставляется в течение 30 секунд (SC-009)

---

### ConfirmPasswordReset

**RPC:** `ConfirmPasswordReset(ConfirmResetPasswordRequest) returns (ConfirmResetPasswordResponse)`  
**HTTP:** `POST /auth/password-reset/confirm`  
**FR:** FR-008, FR-286

Установка нового пароля используя токен сброса.

**Request:**

```protobuf
message ConfirmResetPasswordRequest {
  string token
  string new_password  // те же требования что при регистрации
}
```

**Response:**

```protobuf
message ConfirmResetPasswordResponse {
  string message
}
```

**Требования:**

- Валидация токена сброса
- Применение тех же требований к паролю что при регистрации (FR-073, FR-298)
- Минимум 12 символов (FR-070)
- Проверка через Have I Been Pwned API (FR-071)
- Отклонение скомпрометированных паролей (FR-072)
- Аннулирование всех существующих сессий пользователя

**Ошибки:**

- Токен недействителен или истек
- Новый пароль не соответствует требованиям
- Пароль скомпрометирован

---

### ChangePassword

**RPC:** `ChangePassword(ChangePasswordRequest) returns (ChangePasswordResponse)`  
**HTTP:** `POST /auth/change-password`  
**FR:** FR-287, FR-297, FR-298

Смена пароля аутентифицированным пользователем.

**Request:**

```protobuf
message ChangePasswordRequest {
  string old_password
  string new_password
}
```

**Response:**

```protobuf
message ChangePasswordResponse {
  string message
}
```

**Требования:**

- Валидация старого пароля перед разрешением смены (FR-297)
- Применение тех же требований к новому паролю (FR-298)
- Минимум 12 символов (FR-070)
- Проверка через Have I Been Pwned API (FR-071-072)
- Требуется аутентификация

**Ошибки:**

- Старый пароль неверен
- Новый пароль не соответствует требованиям
- Новый пароль скомпрометирован

---

### GetCurrentSession

**RPC:** `GetCurrentSession(GetCurrentSessionRequest) returns (GetCurrentSessionResponse)`  
**HTTP:** `GET /auth/session`  
**FR:** FR-288, FR-299

Получение информации о текущей сессии.

**Request:**

```protobuf
message GetCurrentSessionRequest {}
```

**Response:**

```protobuf
message GetCurrentSessionResponse {
  Session session
}
```

**Требования:**

- Возврат информации о текущей сессии (FR-299):
  - session_id
  - device_info
  - ip_address
  - last_activity
  - created_at
- Требуется аутентификация

---

### ListActiveSessions

**RPC:** `ListActiveSessions(ListActiveSessionsRequest) returns (ListActiveSessionsResponse)`  
**HTTP:** `GET /auth/sessions`  
**FR:** FR-289, FR-300

Получение списка всех активных сессий пользователя.

**Request:**

```protobuf
message ListActiveSessionsRequest {
  string cursor
  int32 limit
}
```

**Response:**

```protobuf
message ListActiveSessionsResponse {
  repeated Session sessions
  string next_cursor
  bool has_more
}
```

**Требования:**

- Cursor-based пагинация
- Сортировка по last_activity в обратном порядке (новые первые) (FR-300)
- Требуется аутентификация

---

### RevokeSession

**RPC:** `RevokeSession(RevokeSessionRequest) returns (RevokeSessionResponse)`  
**HTTP:** `DELETE /auth/sessions/{session_id}`  
**FR:** FR-290, FR-301

Принудительное завершение конкретной сессии.

**Request:**

```protobuf
message RevokeSessionRequest {
  string session_id
}
```

**Response:**

```protobuf
message RevokeSessionResponse {
  string message
}
```

**Требования:**

- Пользователь может отозвать любую свою сессию кроме текущей (FR-301)
- Аннулирование соответствующего refresh token
- Требуется аутентификация

**Ошибки:**

- Сессия не найдена
- Попытка отозвать текущую сессию
- Сессия принадлежит другому пользователю

---

## Механизм JWT токенов

### Access Token

- **Срок действия:** 15 минут (FR-057)
- **Назначение:** Авторизация запросов
- **Передача:** gRPC Metadata, заголовок Authorization, формат Bearer token (FR-086)
- **Claims:**
  - user_id
  - username
  - is_verified
  - exp (expiration time)
  - iat (issued at)

### Refresh Token

- **Срок действия:** 7 дней (FR-058)
- **Назначение:** Обновление access token
- **Хранение:** Клиент (в безопасном хранилище)
- **Claims:**
  - user_id
  - session_id
  - exp
  - iat

### Валидация токенов

- Валидация через gRPC interceptors (FR-087)
- Проверка подписи
- Проверка срока действия
- Извлечение user identity из claims (FR-089)
- gRPC ошибка Unauthenticated (код 16) при невалидном токене (FR-088)

## Rate Limiting

### Login операции

- **Лимит:** 5 попыток за 10 минут на IP адрес (FR-055, FR-293)
- **Ответ:** HTTP 429 Too Many Requests (FR-059)
- **Сброс:** Автоматически через 10 минут

### API запросы

- **Лимит:** 100 запросов в минуту на аутентифицированного пользователя (FR-056)
- **Ответ:** HTTP 429 Too Many Requests (FR-059)

## Безопасность паролей

### Требования

- Минимальная длина: 12 символов (FR-070)
- Дополнительных требований сложности нет

### Проверка компрометации

- Интеграция с Have I Been Pwned API (FR-071)
- Проверка при регистрации (FR-291)
- Проверка при смене пароля (FR-298)
- Проверка при сбросе пароля (FR-073)
- Четкое сообщение об ошибке при обнаружении (FR-072)

### Хранение

- Хеширование с использованием bcrypt или argon2
- Уникальная соль для каждого пароля
- Никогда не логировать пароли

## Email верификация

### Процесс

1. При регистрации генерируется уникальный токен
2. Токен отправляется на email пользователя в ссылке
3. Пользователь кликает по ссылке
4. Backend валидирует токен и маркирует email как верифицированный

### Ограничения неверифицированных пользователей

Неверифицированные пользователи НЕ могут (FR-009):

- Создавать сообщества
- Создавать посты
- Создавать комментарии

Неверифицированные пользователи МОГУТ:

- Просматривать контент
- Входить в систему

## Управление сессиями

### Создание сессии

При успешном login создается новая сессия с:

- Уникальным session_id
- Информацией об устройстве (User-Agent)
- IP адресом
- Timestamp создания

### Отслеживание активности

- Обновление last_activity при каждом запросе
- Используется для расчета активных пользователей (FR-326)

### Множественные сессии

- Пользователь может иметь несколько одновременных сессий
- Каждая сессия имеет свой refresh token
- Пользователь может просматривать и управлять всеми сессиями

### Автоматическое завершение

- Access token истекает через 15 минут
- Refresh token истекает через 7 дней
- При logout текущая сессия завершается
- При смене пароля ВСЕ сессии завершаются
