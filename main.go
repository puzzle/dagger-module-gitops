package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

const WorkDir = "/gitops"

type PitcGitops struct {
}

// pitc-cicd-helm-demo-prod
func (m *PitcGitops) UpdateHelmRevision(ctx context.Context, gitDir *Directory, envName string, revision string) (*Directory, error) {

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

func (m *PitcGitops) UpdateImageTagHelm(ctx context.Context, gitDir *Directory, valuesFile string, jsonPath string, revision string) (*Directory, error) {

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

func (m *PitcGitops) WithAPI(ctx context.Context, apiUrl string, accessToken string) *MergeRequest {
	return &MergeRequest{
		AccessToken: accessToken,
		ApiUrl:      apiUrl,
	}
}

func (m *MergeRequest) withMergeRequest(ctx context.Context, projectPath string, sourceBranch string, targetBranch string, title Optional[string], descripton Optional[string]) *MergeRequest {

	m.Title = title.GetOr("Dagger Bot MR")
	m.Description = descripton.GetOr("No description provided")
	m.SourceBranch = sourceBranch
	m.TargetBranch = targetBranch
	m.ProjectPath = projectPath

	return m
}

func (m *MergeRequest) createGitLabMR(ctx context.Context) error {

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

func StringPtr(s string) *string {
	return &s
}

type Config struct {
	MrConfig     *MrConfig    `yaml:"ops"`
	HelmPushOpts HelmPushOpts `yaml:"helm"`
}
type MrConfig struct {
	OpsRepository string   `yaml:"opsRepository"`
	Environment   string   `yaml:"environment"`
	Tags          []string `yaml:"tags"`
}

type HelmPushOpts struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	Oci        bool   `yaml:"oci"`
	Username   string `yaml:"username"`
}

func (m *PitcGitops) Run(ctx context.Context, key *File, apiToken string, helmChart *Directory, registryPassword string) error {

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

	config := &Config{}
	err = yaml.Unmarshal(content, config)
	if err != nil {
		return err
	}

	version, err := dag.Helm().Version(ctx, helmChart)
	if err != nil {
		return err
	}

	pushed, err := dag.Helm().PackagePush(ctx, helmChart, config.HelmPushOpts.Registry, config.MrConfig.OpsRepository, config.HelmPushOpts.Username, registryPassword)
	if err != nil {
		return err
	}

	if !pushed {
		fmt.Println("chart is up to date")
		return nil
	}

	gitAction := dag.GitActions().WithRepository(config.MrConfig.OpsRepository, key)
	gitDir := gitAction.CloneSSH()

	//rand := randomstring.HumanFriendlyEnglishString(6)
	prBranch := Opt[string](fmt.Sprintf("update/helm-revision-%s", version))

	gitDir, err = m.
		UpdateHelmRevision(ctx, gitDir, config.MrConfig.Environment, version)
	if err != nil {
		return err
	}

	_, err = gitAction.Push(ctx, gitDir, GitActionsGitActionRepositoryPushOpts{PrBranch: fmt.Sprintf("update/helm-revision-%s", version)})
	if err != nil {
		return err
	}

	return m.WithAPI(ctx, "https://gitlab.puzzle.ch", apiToken).
		withMergeRequest(ctx, "cschlatter/clone-test", prBranch.value, "main", Opt[string](fmt.Sprintf("Update Helm Chart version => %s", version)), Opt[string]("Triggered by Dagger")).
		createGitLabMR(ctx)
}
