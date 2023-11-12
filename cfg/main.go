package cfg

const WorkDir = "/gitops"

type Config struct {
	MrConfig     *MrConfig    `yaml:"ops"`
	HelmPushOpts HelmPushOpts `yaml:"helm"`
}
type MrConfig struct {
	OpsRepository string                 `yaml:"repository"`
	TargetBranch  string                 `yaml:"targetBranch"`
	Environments  map[string]Environment `yaml:"environments"`
	Tags          []string               `yaml:"tags"`
}

type Environment struct {
	//Push into target branch without MR
	Direct bool     `yaml:"direct"`
	Tags   []string `yaml:"tags"`
}

type HelmPushOpts struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	Oci        bool   `yaml:"oci"`
	Username   string `yaml:"username"`
}
