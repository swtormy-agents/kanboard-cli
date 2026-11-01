package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/swtormy-agents/kanboard-cli/internal/kanboard"
)

var rpcMethodName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)

var rpcReadPrefixes = []string{"get", "search", "count", "is", "has", "check"}

func rpcMethodReadOnly(method string) bool {
	for _, p := range rpcReadPrefixes {
		if strings.HasPrefix(method, p) {
			return true
		}
	}
	return false
}

func validateRPCMethod(method string) error {
	if !rpcMethodName.MatchString(method) {
		return usageErr("rpc method %q must match ^[a-zA-Z][a-zA-Z0-9]*$", method)
	}
	return nil
}

func (a *app) rpcCmd() *cobra.Command {
	var paramsJSON string
	var writeFlag bool
	cmd := &cobra.Command{
		Use:   "rpc <method>",
		Short: "Call a Kanboard JSON-RPC method (escape hatch)",
		Long:  "Methods starting with get/search/count/is/has/check are read-only. Other methods need --write and interactive confirmation. Batch requests are not supported.",
		Example: `  kanboard rpc getAllProjects
  kanboard rpc searchTasks --params '{"project_id":1,"query":"status:open"}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := args[0]
			if err := validateRPCMethod(method); err != nil {
				return err
			}
			var params any
			if paramsJSON != "" {
				trim := strings.TrimSpace(paramsJSON)
				if len(trim) > 0 && trim[0] == '[' {
					return usageErr("batch JSON-RPC requests are not supported")
				}
				if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
					return usageErr("--params: %v", err)
				}
			}
			if !rpcMethodReadOnly(method) {
				if err := a.requireHumanWrite(cmd, "rpc "+method); err != nil {
					return err
				}
			}
			return a.rpcCall(cmd, method, params)
		},
	}
	cmd.Flags().StringVar(&paramsJSON, "params", "", "JSON object of named RPC parameters")
	bindWriteFlag(cmd, &writeFlag)
	return cmd
}

func (a *app) healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Verify KANBOARD_URL and API credentials (getVersion)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := a.client(ctx)
			if err != nil {
				return err
			}
			if a.dryRun {
				return a.emitValue(map[string]any{
					"request": kanboard.DryRun("getVersion", c.RPCURL, nil),
					"mode":    "read-only",
				})
			}
			raw, err := c.Call(ctx, "getVersion", nil)
			if err != nil {
				return a.wrapCall(err)
			}
			var server json.RawMessage = raw
			base := a.baseURL
			if base == "" {
				base = strings.TrimSpace(a.getenv("KANBOARD_URL"))
			}
			safeBase, _ := kanboard.ValidateBaseURL(base)
			return a.emitValue(map[string]any{"ok": true, "url": safeBase, "server": server})
		},
	}
}

func (a *app) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Kanboard server version (getVersion)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.rpcCall(cmd, "getVersion", nil)
		},
	}
}

func (a *app) meCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Current user profile (getMe; personal API only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.resolveAuth(); err != nil {
				return err
			}
			if !a.isUserAuth() {
				return usageErr("me requires personal API credentials (set KANBOARD_USERNAME with KANBOARD_API_TOKEN; application API user jsonrpc cannot call getMe)")
			}
			return a.rpcCall(cmd, "getMe", nil)
		},
	}
}

func (a *app) projectsCmd() *cobra.Command {
	projects := &cobra.Command{Use: "projects", Short: "Projects"}
	list := &cobra.Command{
		Use:   "list",
		Short: "List projects (getAllProjects)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.rpcCall(cmd, "getAllProjects", nil)
		},
	}
	projects.AddCommand(list)
	return projects
}

func (a *app) projectCmd() *cobra.Command {
	var projectID string
	get := &cobra.Command{
		Use:   "get",
		Short: "Get a project (getProjectById)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePositiveID("--id", projectID)
			if err != nil {
				return err
			}
			return a.rpcCall(cmd, "getProjectById", idParam("project_id", id))
		},
	}
	get.Flags().StringVar(&projectID, "id", "", "project id (required)")
	_ = get.MarkFlagRequired("id")
	project := &cobra.Command{Use: "project", Short: "Project"}
	project.AddCommand(get)
	return project
}

func (a *app) columnsCmd() *cobra.Command {
	columns := &cobra.Command{Use: "columns", Short: "Board columns"}
	var project string
	list := &cobra.Command{
		Use:   "list",
		Short: "List columns (getColumns)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePositiveID("--project", project)
			if err != nil {
				return err
			}
			return a.rpcCall(cmd, "getColumns", idParam("project_id", id))
		},
	}
	list.Flags().StringVar(&project, "project", "", "project id (required)")
	_ = list.MarkFlagRequired("project")
	columns.AddCommand(list)
	return columns
}

func (a *app) swimlanesCmd() *cobra.Command {
	swimlanes := &cobra.Command{Use: "swimlanes", Short: "Swimlanes"}
	var project string
	list := &cobra.Command{
		Use:   "list",
		Short: "List swimlanes (getAllSwimlanes)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePositiveID("--project", project)
			if err != nil {
				return err
			}
			return a.rpcCall(cmd, "getAllSwimlanes", idParam("project_id", id))
		},
	}
	list.Flags().StringVar(&project, "project", "", "project id (required)")
	_ = list.MarkFlagRequired("project")
	swimlanes.AddCommand(list)
	return swimlanes
}

func (a *app) tasksCmd() *cobra.Command {
	tasks := &cobra.Command{Use: "tasks", Short: "Tasks", Aliases: []string{"task"}}
	var project, getTaskID, moveProject, moveTaskID, closeTaskID, openTaskID, status string
	var writeFlag bool

	list := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a project (getAllTasks)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := parsePositiveID("--project", project)
			if err != nil {
				return err
			}
			statusID, err := parseTaskStatus(status)
			if err != nil {
				return err
			}
			params := map[string]any{"project_id": pid, "status_id": statusID}
			return a.rpcCall(cmd, "getAllTasks", params)
		},
	}
	list.Flags().StringVar(&project, "project", "", "project id (required)")
	list.Flags().StringVar(&status, "status", "active", "active or inactive")
	_ = list.MarkFlagRequired("project")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a task (getTask)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePositiveID("--id", getTaskID)
			if err != nil {
				return err
			}
			return a.rpcCall(cmd, "getTask", idParam("task_id", id))
		},
	}
	get.Flags().StringVar(&getTaskID, "id", "", "task id (required)")
	_ = get.MarkFlagRequired("id")

	create := a.taskCreateCmd(&writeFlag)
	move := a.taskMoveCmd(&writeFlag, &moveProject, &moveTaskID)
	closeCmd := a.taskCloseOpenCmd("close", "closeTask", &writeFlag, &closeTaskID)
	openCmd := a.taskCloseOpenCmd("open", "openTask", &writeFlag, &openTaskID)

	tasks.AddCommand(list, get, create, move, closeCmd, openCmd)
	return tasks
}

func (a *app) taskCreateCmd(writeFlag *bool) *cobra.Command {
	var project, title, description, column, swimlane, owner, color, due string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a task (createTask)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireHumanWrite(cmd, "task create"); err != nil {
				return err
			}
			pid, err := parsePositiveID("--project", project)
			if err != nil {
				return err
			}
			if strings.TrimSpace(title) == "" {
				return usageErr("--title is required")
			}
			params := map[string]any{
				"title":      title,
				"project_id": pid,
			}
			if description != "" {
				params["description"] = description
			}
			if color != "" {
				params["color_id"] = color
			}
			if due != "" {
				params["date_due"] = due
			}
			if column != "" {
				colID, err := parsePositiveID("--column", column)
				if err != nil {
					return err
				}
				params["column_id"] = colID
			}
			if swimlane != "" {
				swID, err := parsePositiveID("--swimlane", swimlane)
				if err != nil {
					return err
				}
				params["swimlane_id"] = swID
			}
			if owner != "" {
				ownerID, err := parsePositiveID("--owner", owner)
				if err != nil {
					return err
				}
				params["owner_id"] = ownerID
			}
			return a.rpcCall(cmd, "createTask", params)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project id (required)")
	cmd.Flags().StringVar(&title, "title", "", "task title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Markdown description")
	cmd.Flags().StringVar(&column, "column", "", "column id")
	cmd.Flags().StringVar(&swimlane, "swimlane", "", "swimlane id")
	cmd.Flags().StringVar(&owner, "owner", "", "owner user id")
	cmd.Flags().StringVar(&color, "color", "", "color id (e.g. yellow, blue)")
	cmd.Flags().StringVar(&due, "due", "", "due date (YYYY-MM-DD HH:MM)")
	bindWriteFlag(cmd, writeFlag)
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func (a *app) taskMoveCmd(writeFlag *bool, project, taskID *string) *cobra.Command {
	var column, position, swimlane string
	cmd := &cobra.Command{
		Use:   "move",
		Short: "Move a task on the board (moveTaskPosition)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireHumanWrite(cmd, "task move"); err != nil {
				return err
			}
			pid, err := parsePositiveID("--project", *project)
			if err != nil {
				return err
			}
			tid, err := parsePositiveID("--id", *taskID)
			if err != nil {
				return err
			}
			colID, err := parsePositiveID("--column", column)
			if err != nil {
				return err
			}
			posID, err := parsePositiveID("--position", position)
			if err != nil {
				return err
			}
			swID, err := parsePositiveID("--swimlane", swimlane)
			if err != nil {
				return err
			}
			params := map[string]any{
				"project_id":  pid,
				"task_id":     tid,
				"column_id":   colID,
				"position":    posID,
				"swimlane_id": swID,
			}
			return a.rpcCall(cmd, "moveTaskPosition", params)
		},
	}
	cmd.Flags().StringVar(project, "project", "", "project id (required)")
	cmd.Flags().StringVar(taskID, "id", "", "task id (required)")
	cmd.Flags().StringVar(&column, "column", "", "destination column id (required)")
	cmd.Flags().StringVar(&position, "position", "1", "position within the column")
	cmd.Flags().StringVar(&swimlane, "swimlane", "1", "destination swimlane id")
	bindWriteFlag(cmd, writeFlag)
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("column")
	return cmd
}

func (a *app) taskCloseOpenCmd(use, method string, writeFlag *bool, taskID *string) *cobra.Command {
	short := fmt.Sprintf("%s a task (%s)", use, method)
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireHumanWrite(cmd, "task "+use); err != nil {
				return err
			}
			id, err := parsePositiveID("--id", *taskID)
			if err != nil {
				return err
			}
			return a.rpcCall(cmd, method, idParam("task_id", id))
		},
	}
	cmd.Flags().StringVar(taskID, "id", "", "task id (required)")
	bindWriteFlag(cmd, writeFlag)
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func (a *app) commentsCmd() *cobra.Command {
	comments := &cobra.Command{Use: "comments", Short: "Task comments"}
	var listTaskID, addTaskID, text string
	var writeFlag bool

	list := &cobra.Command{
		Use:   "list",
		Short: "List comments on a task (getAllComments)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePositiveID("--task", listTaskID)
			if err != nil {
				return err
			}
			return a.rpcCall(cmd, "getAllComments", idParam("task_id", id))
		},
	}
	list.Flags().StringVar(&listTaskID, "task", "", "task id (required)")
	_ = list.MarkFlagRequired("task")

	add := &cobra.Command{
		Use:   "add",
		Short: "Add a comment (createComment)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireHumanWrite(cmd, "comment add"); err != nil {
				return err
			}
			tid, err := parsePositiveID("--task", addTaskID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(text) == "" {
				return usageErr("--text is required")
			}
			userID := uint64(0)
			if err := a.resolveAuth(); err != nil {
				return err
			}
			if a.isUserAuth() {
				ctx := cmd.Context()
				c, err := a.client(ctx)
				if err != nil {
					return err
				}
				meRaw, err := c.Call(ctx, "getMe", nil)
				if err != nil {
					return a.wrapCall(err)
				}
				var me map[string]any
				if err := json.Unmarshal(meRaw, &me); err != nil {
					return runtimeErr(err)
				}
				switch v := me["id"].(type) {
				case float64:
					userID = uint64(v)
				case json.Number:
					n, _ := v.Int64()
					userID = uint64(n)
				case string:
					n, err := parsePositiveID("user id", v)
					if err != nil {
						return err
					}
					userID = n
				}
			}
			params := map[string]any{
				"task_id": tid,
				"user_id": userID,
				"content": text,
			}
			return a.rpcCall(cmd, "createComment", params)
		},
	}
	add.Flags().StringVar(&addTaskID, "task", "", "task id (required)")
	add.Flags().StringVar(&text, "text", "", "comment text (required)")
	bindWriteFlag(add, &writeFlag)
	_ = add.MarkFlagRequired("task")
	_ = add.MarkFlagRequired("text")

	comments.AddCommand(list, add)
	return comments
}

func (a *app) commentCmd() *cobra.Command {
	var taskID, text string
	var writeFlag bool
	add := &cobra.Command{
		Use:   "add",
		Short: "Add a comment (createComment)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.requireHumanWrite(cmd, "comment add"); err != nil {
				return err
			}
			tid, err := parsePositiveID("--task", taskID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(text) == "" {
				return usageErr("--text is required")
			}
			userID := uint64(0)
			if err := a.resolveAuth(); err != nil {
				return err
			}
			if a.isUserAuth() {
				ctx := cmd.Context()
				c, err := a.client(ctx)
				if err != nil {
					return err
				}
				meRaw, err := c.Call(ctx, "getMe", nil)
				if err != nil {
					return a.wrapCall(err)
				}
				var me map[string]any
				if err := json.Unmarshal(meRaw, &me); err != nil {
					return runtimeErr(err)
				}
				switch v := me["id"].(type) {
				case float64:
					userID = uint64(v)
				case json.Number:
					n, _ := v.Int64()
					userID = uint64(n)
				case string:
					n, err := parsePositiveID("user id", v)
					if err != nil {
						return err
					}
					userID = n
				}
			}
			params := map[string]any{
				"task_id": tid,
				"user_id": userID,
				"content": text,
			}
			return a.rpcCall(cmd, "createComment", params)
		},
	}
	add.Flags().StringVar(&taskID, "task", "", "task id (required)")
	add.Flags().StringVar(&text, "text", "", "comment text (required)")
	bindWriteFlag(add, &writeFlag)
	_ = add.MarkFlagRequired("task")
	_ = add.MarkFlagRequired("text")
	comment := &cobra.Command{Use: "comment", Short: "Task comment"}
	comment.AddCommand(add)
	return comment
}

func (a *app) usersCmd() *cobra.Command {
	users := &cobra.Command{Use: "users", Short: "Users (metadata only)"}
	list := &cobra.Command{
		Use:   "list",
		Short: "List users (getAllUsers; id, username, name only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := a.client(ctx)
			if err != nil {
				return err
			}
			if a.dryRun {
				return a.emitValue(kanboard.DryRun("getAllUsers", c.RPCURL, nil))
			}
			raw, err := c.Call(ctx, "getAllUsers", nil)
			if err != nil {
				return a.wrapCall(err)
			}
			safe, err := kanboard.SanitizeUsers(raw)
			if err != nil {
				return runtimeErr(err)
			}
			return a.emit(safe)
		},
	}
	users.AddCommand(list)
	return users
}

func (a *app) searchCmd() *cobra.Command {
	var project, query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search tasks (searchTasks)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := parsePositiveID("--project", project)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return usageErr("--query is required")
			}
			params := map[string]any{"project_id": pid, "query": query}
			return a.rpcCall(cmd, "searchTasks", params)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project id (required)")
	cmd.Flags().StringVar(&query, "query", "", "Kanboard search query (required)")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}
