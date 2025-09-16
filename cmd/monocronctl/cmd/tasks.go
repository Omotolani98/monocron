package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
    Use:   "tasks",
    Short: "Manage tasks",
}

var tasksDeleteCmd = &cobra.Command{
    Use:   "delete <name>",
    Short: "Delete a task",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]
        if err := del(apiBase+"/tasks/"+name); err != nil { return err }
        fmt.Println("deleted", name)
        return nil
    },
}

func init() {
    tasksCmd.AddCommand(tasksDeleteCmd)
}

