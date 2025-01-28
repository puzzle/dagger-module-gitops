package main

import (
	"context"
	"main/internal/dagger"
	"fmt"
	"main/cfg"
	"os"
	"strings"

	giturls "github.com/whilp/git-urls"
	"github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

const WorkDir = "/gitops"

type PitcGitops struct {
}

// pitc-cicd-helm-demo-prod
func (m *PitcGitops) UpdateHelmRevision(
	ctx context.Context,
	gitDir *dagger.Directory,
	envName string,
	revision string,
) (*dagger.Directory, error) {

	mod := dag.Container().From("registry.puzzle.ch/cicd/alpine-base:latest").
		WithDirectory(WorkDir, gitDir).
		WithWorkdir(WorkDir).
		//WithExec([]string{"git", "switch", "-c", pushBranch.GetOr("main")}).
		WithExec([]string{"sh", "-c", fmt.Sprintf("yq eval '.environments | map(select(.name == \"%s\")).[].argocd.helm.targetRevision'  argocd/values.yaml", envName)})

	deployVersion, err := mod.Stdout(ctx)
	if err != nil {
		return nil, err
	}

	if revision == strings.TrimSpace(deployVersion) {
		fmt.Printf("skip task, version %s already deployed\n", deployVersion)
		return gitDir, nil
	}

	return mod.WithExec([]string{"sh", "-c", fmt.Sprintf("yq eval '.environments |= map(select(.name == \"%s\").argocd.helm.targetRevision=\"%s\")' -i argocd/values.yaml", envName, revision)}).
		Directory(WorkDir).Sync(ctx)

}

func (m *PitcGitops) UpdateImageTagHelm(
	ctx context.Context,
	gitDir *dagger.Directory,
	valuesFile string,
	jsonPath string,
	revision string,
) (*dagger.Directory, error) {

	return dag.Container().From("registry.puzzle.ch/cicd/alpine-base:latest").
		WithDirectory(WorkDir, gitDir).
		WithWorkdir(WorkDir).
		WithExec([]string{"sh", "-c", fmt.Sprintf("yq eval '%s=\"%s\"' -i %s", jsonPath, revision, valuesFile)}).
		Directory(WorkDir).Sync(ctx)

}

type MergeRequest struct {
	Title        string
	Description  string
	SourceBranch string
	TargetBranch string
	ProjectPath  string
	ApiUrl       string
	AccessToken  string
}

func (m *PitcGitops) WithAPI(
	ctx context.Context,
	apiUrl string,
	accessToken string,
) *MergeRequest {
	return &MergeRequest{
		AccessToken: accessToken,
		ApiUrl:      apiUrl,
	}
}

func (m *MergeRequest) withMergeRequest(
	ctx context.Context,
	projectPath string,
	sourceBranch string,
	targetBranch string,
	// +optional
	title string,
	// +optional
	description string,
	tags []string,
) *MergeRequest {

	if title != "" {
		m.Title = title
	} else {
		m.Title = "Dagger Bot MR"
	}
	if description != "" {
		m.Description = description
	} else {
		m.Description = "No description provided"
	}
	m.SourceBranch = sourceBranch
	m.TargetBranch = targetBranch
	m.ProjectPath = projectPath

	return m
}

func (m *MergeRequest) createGitLabMR(
	ctx context.Context,
) error {

	glClient, err := gitlab.NewClient(m.AccessToken, gitlab.WithBaseURL(m.ApiUrl))
	if err != nil {
		return err
	}

	_, _, err = glClient.MergeRequests.CreateMergeRequest(m.ProjectPath, &gitlab.CreateMergeRequestOptions{
		Title:        &m.Title,
		Description:  &m.Description,
		SourceBranch: &m.SourceBranch,
		TargetBranch: &m.TargetBranch,
		Labels:       &gitlab.Labels{"auto"},
	})

	return err
}

func StringPtr(
	s string,
) *string {
	return &s
}

func (m *PitcGitops) Run(
	ctx context.Context,
	key *dagger.File,
	apiToken string,
	helmChart *dagger.Directory,
	registryPassword *dagger.Secret,
) error {

	_, err := os.Stat("./ci.yaml")
	if err != nil {
		//No config provided
		fmt.Println("No config provided, skip task")
		return nil
	}

	content, err := os.ReadFile("./ci.yaml")
	if err != nil {
		return err
	}

	config := &cfg.Config{}
	err = yaml.Unmarshal(content, config)
	if err != nil {
		return err
	}

	version, err := dag.Helm().Version(ctx, helmChart)
	if err != nil {
		return err
	}

	pushed, err := dag.Helm().PackagePush(ctx, helmChart, config.HelmPushOpts.Registry, config.HelmPushOpts.Repository, config.HelmPushOpts.Username, registryPassword)
	if err != nil {
		return err
	}

	if !pushed {
		fmt.Println("chart is up to date")
		return nil
	}

	gitAction := dag.GitActions().WithRepository(config.MrConfig.OpsRepository, key)
	gitDir := gitAction.CloneSSH()

	for name, env := range config.MrConfig.Environments {

		fmt.Println("process environemnt: " + name)

		var prBranch string
		if env.Direct {
			prBranch = config.MrConfig.TargetBranch
		} else {
			prBranch = fmt.Sprintf("update/helm-revision-%s", version)
		}

		gitDir, err = m.
			UpdateHelmRevision(ctx, gitDir, name, version)
		if err != nil {
			return err
		}

		err = gitAction.Push(ctx, gitDir, dagger.GitActionsGitActionRepositoryPushOpts{PrBranch: prBranch})
		if err != nil {
			return err
		}

		if env.Direct {
			fmt.Println("push direct, skip mr")
			return nil
		}

		url, err := giturls.Parse(config.MrConfig.OpsRepository)
		if err != nil {
			return err
		}

		project := strings.TrimSuffix(url.Host, ".git")

		tags := append(config.MrConfig.Tags, env.Tags...)

		err = m.WithAPI(ctx, "https://gitlab.puzzle.ch", apiToken).
			withMergeRequest(ctx, project, prBranch, config.MrConfig.TargetBranch, fmt.Sprintf("Update Helm Chart version => %s", version), "Triggered by Dagger", tags).
			createGitLabMR(ctx)

		if err != nil {
			return err
		}
	}

	return nil
}
