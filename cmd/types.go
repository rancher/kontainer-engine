package cmd

type KubeConfig struct {
	APIVersion     string          `yaml:"apiVersion,omitempty"`
	Clusters       []ConfigCluster `yaml:"clusters,omitempty"`
	Contexts       []ConfigContext `yaml:"contexts,omitempty"`
	Users          []ConfigUser    `yaml:"users,omitempty"`
	CurrentContext string          `yaml:"current-context,omitempty"`
	Kind           string          `yaml:"kind,omitempty"`
	Preferences    string          `yaml:"preferences,omitempty"`
}

type ConfigCluster struct {
	Cluster DataCluster `yaml:"cluster,omitempty"`
	Name    string      `yaml:"name,omitempty"`
}

type DataCluster struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	Server                   string `yaml:"server,omitempty"`
}

type ConfigContext struct {
	Context ContextData `yaml:"context,omitempty"`
	Name    string      `yaml:"name,omitempty"`
}

type ContextData struct {
	Cluster string `yaml:"cluster,omitempty"`
	User    string `yaml:"user,omitempty"`
}

type ConfigUser struct {
	Name string   `yaml:"name,omitempty"`
	User UserData `yaml:"user,omitempty"`
}

type UserData struct {
	Token    string `yaml:"token,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}
