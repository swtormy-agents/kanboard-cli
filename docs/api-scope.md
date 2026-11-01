# Kanboard API scope (kanboard-cli v0.1)

This document lists JSON-RPC methods used by **kanboard-cli v0.1**. The CLI is unofficial and community-built; it is not affiliated with Kanboard.

Authentication for all calls: HTTP Basic (application API user `jsonrpc` + `KANBOARD_API_TOKEN`, or personal API `KANBOARD_USERNAME` + `KANBOARD_API_TOKEN`). Base URL from env `KANBOARD_URL` or flag `--url`; requests go to `<base>/jsonrpc.php` unless the URL already ends with `/jsonrpc.php`.

Official docs: [Kanboard API reference](https://docs.kanboard.org/v1/api/). Source procedures: [kanboard/kanboard `app/Api/Procedure/`](https://github.com/kanboard/kanboard/tree/main/app/Api/Procedure).

## Covered in v0.1

| Command | JSON-RPC method | Params (named) | Docs / source |
|---------|-----------------|----------------|---------------|
| `kanboard health`, `kanboard version` | `getVersion` | _(none)_ | [Application API](https://docs.kanboard.org/v1/api/application_api/) · `app/Api/Procedure/AppProcedure.php` |
| `kanboard me` | `getMe` | _(none)_ | [Current User API](https://docs.kanboard.org/v1/api/me/) · `app/Api/Procedure/MeProcedure.php` |
| `kanboard projects list` | `getAllProjects` | _(none)_ | [Project procedures](https://docs.kanboard.org/v1/api/project/) · `app/Api/Procedure/ProjectProcedure.php` |
| `kanboard project get --id` | `getProjectById` | `project_id` | [Project procedures](https://docs.kanboard.org/v1/api/project/) · `ProjectProcedure.php` |
| `kanboard columns list --project` | `getColumns` | `project_id` | [Column procedures](https://docs.kanboard.org/v1/api/column/) · `app/Api/Procedure/ColumnProcedure.php` |
| `kanboard swimlanes list --project` | `getAllSwimlanes` | `project_id` | [Swimlane procedures](https://docs.kanboard.org/v1/api/swimlane/) · `app/Api/Procedure/SwimlaneProcedure.php` |
| `kanboard tasks list --project [--status]` | `getAllTasks` | `project_id`, `status_id` (1 active, 0 inactive) | [Task procedures](https://docs.kanboard.org/v1/api/task/) · `app/Api/Procedure/TaskProcedure.php` |
| `kanboard task get --id` | `getTask` | `task_id` | [Task procedures](https://docs.kanboard.org/v1/api/task/) · `TaskProcedure.php` |
| `kanboard comments list --task` | `getAllComments` | `task_id` | [Comment procedures](https://docs.kanboard.org/v1/api/comment/) · `app/Api/Procedure/CommentProcedure.php` |
| `kanboard users list` | `getAllUsers` | _(none)_ — response stripped to `id`, `username`, `name` | [User procedures](https://docs.kanboard.org/v1/api/user/) · `app/Api/Procedure/UserProcedure.php` |
| `kanboard search --project --query` | `searchTasks` | `project_id`, `query` | [Task procedures](https://docs.kanboard.org/v1/api/task/) · `TaskProcedure.php` |
| `kanboard task create --write …` | `createTask` | `title`, `project_id`, optional `description`, `column_id`, `swimlane_id`, `owner_id`, `color_id`, `date_due`, … | [Task procedures](https://docs.kanboard.org/v1/api/task/) · `TaskProcedure.php` |
| `kanboard task move --write …` | `moveTaskPosition` | `project_id`, `task_id`, `column_id`, `position`, `swimlane_id` | [Task procedures](https://docs.kanboard.org/v1/api/task/) · `TaskProcedure.php` |
| `kanboard task close --write --id` | `closeTask` | `task_id` | [Task procedures](https://docs.kanboard.org/v1/api/task/) · `TaskProcedure.php` |
| `kanboard task open --write --id` | `openTask` | `task_id` | [Task procedures](https://docs.kanboard.org/v1/api/task/) · `TaskProcedure.php` |
| `kanboard comments add --write --task --text` | `createComment` | `task_id`, `user_id`, `content` (`user_id` from `getMe` on personal API) | [Comment procedures](https://docs.kanboard.org/v1/api/comment/) · `CommentProcedure.php` |
| `kanboard rpc <method>` | * | `--params` JSON object | Escape hatch; read-only prefix rule documented in `--help` |

## Out of scope

- JSON-RPC batch requests (array of calls in one HTTP body)
- Returning email addresses from `users list`
- Non-interactive writes (agents must not pass `--write`)
