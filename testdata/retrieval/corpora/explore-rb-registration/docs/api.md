# Users API

## GET /users

Lists users, newest last. Accepts `limit` (default 50).

## POST /users

Registers a new user. Body: `{"user": {"email": "...", "password": "...", "display_name": "..."}}`.
Responds with `201 Created` and the user, or `422` with an `errors` array.
