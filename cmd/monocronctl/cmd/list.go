package cmd

import (
    "fmt"
    "time"

    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List resources (runs, tasks, runners)",
}

var listRunsCmd = &cobra.Command{
    Use:   "runs",
    Short: "List runs",
    RunE: func(cmd *cobra.Command, args []string) error {
        status, _ := cmd.Flags().GetString("status")
        url := apiBase + "/runs"
        if status != "" { url += "?status=" + status }
        var out struct{ Runs []struct {
            ID string `json:"id"`
            TaskName string `json:"task_name"`
            ScheduledAt time.Time `json:"scheduled_at"`
            Status string `json:"status"`
            Source string `json:"source"`
        } `json:"runs"` }
        if err := getJSON(url, &out); err != nil { return err }
        header := lipgloss.NewStyle().Bold(true)
        fmt.Printf("%s  %s  %s  %s  %s\n", header.Render("RunID"), header.Render("Task"), header.Render("ScheduledAt"), header.Render("Status"), header.Render("Source"))
        for _, r := range out.Runs {
            fmt.Printf("%-36s  %-22s  %-25s  %-10s  %-7s\n", r.ID, r.TaskName, r.ScheduledAt.Format(time.RFC3339), r.Status, r.Source)
        }
        return nil
    },
}

var listTasksCmd = &cobra.Command{
    Use:   "tasks",
    Short: "List tasks",
    RunE: func(cmd *cobra.Command, args []string) error {
        var out struct{ Tasks []struct {
            Author string `json:"author"`
            Name string `json:"task"`
            Schedule string `json:"schedule"`
            NextAt time.Time `json:"schedule_at"`
            LastStatus string `json:"status"`
        } `json:"tasks"` }
        if err := getJSON(apiBase+"/tasks", &out); err != nil { return err }
        header := lipgloss.NewStyle().Bold(true)
        fmt.Printf("%s  %s  %s  %s\n", header.Render("Author"), header.Render("Task"), header.Render("ScheduleAt"), header.Render("Status"))
        for _, t := range out.Tasks {
            author := t.Author
            if author == "" { author = "-" }
            fmt.Printf("%-12s  %-24s  %-25s  %-10s\n", author, t.Name, t.NextAt.Format(time.RFC3339), t.LastStatus)
        }
        return nil
    },
}

var listRunnersCmd = &cobra.Command{
    Use:   "runners",
    Short: "List runners",
    RunE: func(cmd *cobra.Command, args []string) error {
        var out struct{ Runners []struct {
            RunnerID string `json:"runner_id"`
            Kind string `json:"kind"`
            Status string `json:"status"`
            LastSeen time.Time `json:"last_seen"`
        } `json:"runners"` }
        if err := getJSON(apiBase+"/runners", &out); err != nil { return err }
        header := lipgloss.NewStyle().Bold(true)
        fmt.Printf("%s  %s  %s  %s\n", header.Render("runner_id"), header.Render("type"), header.Render("status"), header.Render("last_seen"))
        for _, r := range out.Runners {
            fmt.Printf("%-38s  %-10s  %-6s  %-25s\n", r.RunnerID, r.Kind, r.Status, r.LastSeen.Format(time.RFC3339))
        }
        return nil
    },
}

func init() {
    listRunsCmd.Flags().String("status", "", "Filter by status (QUEUED|RUNNING|COMPLETED|FAILED)")
    listCmd.AddCommand(listRunsCmd)
    listCmd.AddCommand(listTasksCmd)
    listCmd.AddCommand(listRunnersCmd)
}

