# Yet another `todoList` server

- **tiny** (just 2.7M)
- http compliant **for real**
- with a **ugly** web page
- no **AI** used

## HTTP endpoints

- `/` serves the index webpage
- `/readyz` replies with `OK`
- GET `/tasks` get tasks
- POST `/tasks` create a task, BODY: `title` `description`
- PUT `/tasks` update a task, QUERY: `id` `completed`
- DELETE `/tasks` delete a task, QUERY: `id`



| METHOD   | PATH     | DESCRIPTION                     | SUCCESS         |
|----------|----------|---------------------------------|-----------------|
| GET      | `/`      | Serve the index web page        | `200 text/html` |
| GET      | `/readyz`| Liveness probe                  | `200` `OK`      |
| GET      | `/tasks` | List tasks                      | `200`           |
| POST     | `/tasks` | Create a task                   | `201`           |
| PUT      | `/tasks` | Update a task's completed state | `200`           |
| DELETE   | `/tasks` | Delete a task                   | `200`           |
| OPTIONS  | any      | Capability discovery            | `204`           |

### GET `/tasks`

Lists every task. The representation follows `Accept`:

- `application/json` (or a media range that prefers it) → JSON array of task objects;
- otherwise → one task per line as `text/plain`;
- `406 Not Acceptable` if the client accepts neither.

```json
[
  {
    "ID": 1,
    "Title": "Buy milk",
    "Description": "2 liters",
    "Completed": false,
    "Date": "2026-10-08 11:35:34"
  }
]
```

The list is always a JSON array (`[]` when empty), never `null`.

### POST `/tasks`

Creates a task. The body is a form (`application/x-www-form-urlencoded` or `multipart/form-data`).

| FIELD         | REQUIRED | DESCRIPTION      |
|---------------|----------|------------------|
| `title`       | yes      | Task title       |
| `description` | no       | Task description |



### PUT `/tasks`

Updates the completed state.

| PARAM       | REQUIRED | DESCRIPTION                                                        |
|-------------|----------|--------------------------------------------------------------------|
| `id`        | yes      | Task ID                                                            |
| `completed` | no       | `true`/`1` or `false`/`0`. When omitted, toggles the current value |


### DELETE `/tasks`

Deletes a task.

| PARAM | REQUIRED | DESCRIPTION |
|-------|----------|-------------|
| `id`  | yes      | Task ID     |

### Errors

Error responses carry the matching status code and a JSON body:

```json
{ "error": "Title is required" }
```

### Examples

```sh
curl localhost:8080/readyz
curl -H 'Accept: application/json' localhost:8080/tasks
curl -X POST -F title='Buy milk' -F description='2 liters' localhost:8080/tasks
curl -X PUT 'localhost:8080/tasks?id=1&completed=true'
curl -X DELETE 'localhost:8080/tasks?id=1'
```

## Configuration

Via environment variables

| NAME    | DESCRIPTION                                        | DEFAULT       |
|---------|----------------------------------------------------|---------------|
| PORT    | Webserver port                                     | 8080          |
| DB_NAME | Sqlite DB name                                     | todoList.db   |
| TZ      | Timezone used for task timestamps (IANA name)      | system / UTC  |

## Usage

### Run locally

```sh
PORT=9000 DB_NAME=tasks.db ./todolist
```
### Docker

```sh
docker run --rm -p 8080:8080 ghcr.io/<owner>/<repo>:latest
```

## License

MIT License

CSS from [botoxparty](https://github.com/botoxparty/XP.css)
