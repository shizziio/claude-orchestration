package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/nextlevelbuilder/goclaw/internal/config"
)

func devflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "devflow",
		Aliases: []string{"df"},
		Short:   "Manage DevFlow projects and runs",
	}
	cmd.AddCommand(devflowListCmd())
	cmd.AddCommand(devflowExecCmd())
	cmd.AddCommand(devflowStartCmd())
	cmd.AddCommand(devflowStopCmd())
	return cmd
}

func devflowListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List DevFlow projects",
		Run: func(cmd *cobra.Command, args []string) {
			body, err := devflowHTTP("GET", "/v1/devflow/projects", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			var projects []struct {
				ID            string  `json:"id"`
				Name          string  `json:"name"`
				Slug          string  `json:"slug"`
				Status        string  `json:"status"`
				DefaultBranch string  `json:"default_branch"`
				RepoURL       *string `json:"repo_url"`
				WorkspacePath *string `json:"workspace_path"`
			}
			if err := json.Unmarshal(body, &projects); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing: %v\n", err)
				os.Exit(1)
			}
			if jsonOutput {
				fmt.Println(string(body))
				return
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "SLUG\tNAME\tSTATUS\tBRANCH\tWORKSPACE")
			for _, p := range projects {
				ws := "-"
				if p.WorkspacePath != nil {
					ws = *p.WorkspacePath
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Slug, p.Name, p.Status, p.DefaultBranch, ws)
			}
			tw.Flush()
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	return cmd
}

func devflowExecCmd() *cobra.Command {
	var branch string
	cmd := &cobra.Command{
		Use:   "exec <project-slug> <task>",
		Short: "Create and start a run for a project",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			slug := args[0]
			task := strings.Join(args[1:], " ")

			// Resolve project by slug
			project, err := devflowResolveProject(slug)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Create run
			runBody := map[string]any{"task_description": task}
			if branch != "" {
				runBody["branch"] = branch
			}
			payload, _ := json.Marshal(runBody)
			body, err := devflowHTTP("POST", fmt.Sprintf("/v1/devflow/projects/%s/runs", project.ID), payload)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating run: %v\n", err)
				os.Exit(1)
			}
			var run struct {
				ID string `json:"id"`
			}
			json.Unmarshal(body, &run)

			// Start run
			_, err = devflowHTTP("POST", fmt.Sprintf("/v1/devflow/projects/%s/runs/%s/start", project.ID, run.ID), nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error starting run: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Run started: %s (project: %s)\n", run.ID, slug)
			fmt.Printf("Task: %s\n", task)
			if branch != "" {
				fmt.Printf("Branch: %s\n", branch)
			}

			// Poll for completion
			fmt.Println("\nPolling for completion...")
			for {
				time.Sleep(5 * time.Second)
				body, err := devflowHTTP("GET", fmt.Sprintf("/v1/devflow/projects/%s/runs/%s", project.ID, run.ID), nil)
				if err != nil {
					continue
				}
				var status struct {
					Status        string  `json:"status"`
					ResultSummary *string `json:"result_summary"`
					ErrorMessage  *string `json:"error_message"`
				}
				json.Unmarshal(body, &status)
				switch status.Status {
				case "completed":
					fmt.Println("✓ Run completed")
					if status.ResultSummary != nil {
						fmt.Printf("\nResult:\n%s\n", *status.ResultSummary)
					}
					return
				case "failed":
					fmt.Println("✗ Run failed")
					if status.ErrorMessage != nil {
						fmt.Printf("\nError:\n%s\n", *status.ErrorMessage)
					}
					os.Exit(1)
				default:
					fmt.Printf("  status: %s\n", status.Status)
				}
			}
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "", "git branch for worktree isolation")
	return cmd
}

func devflowStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <project-slug> <env-slug>",
		Short: "Start Docker Compose for an environment",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			project, err := devflowResolveProject(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			env, err := devflowResolveEnv(project.ID, args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			_, err = devflowHTTP("POST", fmt.Sprintf("/v1/devflow/projects/%s/environments/%s/start", project.ID, env.ID), nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ Environment %s started\n", args[1])
		},
	}
}

func devflowStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <project-slug> <env-slug>",
		Short: "Stop Docker Compose for an environment",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			project, err := devflowResolveProject(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			env, err := devflowResolveEnv(project.ID, args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			_, err = devflowHTTP("POST", fmt.Sprintf("/v1/devflow/projects/%s/environments/%s/stop", project.ID, env.ID), nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ Environment %s stopped\n", args[1])
		},
	}
}

// --- helpers ---

type devflowProject struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type devflowEnv struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

func devflowResolveProject(slug string) (*devflowProject, error) {
	body, err := devflowHTTP("GET", "/v1/devflow/projects", nil)
	if err != nil {
		return nil, err
	}
	var projects []devflowProject
	json.Unmarshal(body, &projects)
	for _, p := range projects {
		if p.Slug == slug {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", slug)
}

func devflowResolveEnv(projectID, envSlug string) (*devflowEnv, error) {
	body, err := devflowHTTP("GET", fmt.Sprintf("/v1/devflow/projects/%s/environments", projectID), nil)
	if err != nil {
		return nil, err
	}
	var envs []devflowEnv
	json.Unmarshal(body, &envs)
	for _, e := range envs {
		if e.Slug == envSlug {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("environment %q not found", envSlug)
}

func mustLoadConfig() *config.Config {
	cfgPath := resolveConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func devflowHTTP(method, path string, payload []byte) ([]byte, error) {
	cfg := mustLoadConfig()
	baseURL := fmt.Sprintf("http://%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	if cfg.Gateway.Host == "0.0.0.0" {
		baseURL = fmt.Sprintf("http://127.0.0.1:%d", cfg.Gateway.Port)
	}

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = strings.NewReader(string(payload))
	}
	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Use first API key or admin token if available
	token := os.Getenv("GOCLAW_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Default tenant for CLI
	req.Header.Set("X-GoClaw-Tenant-Id", "master")
	req.Header.Set("X-GoClaw-User-Id", "system")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to gateway at %s — is it running?", baseURL)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
