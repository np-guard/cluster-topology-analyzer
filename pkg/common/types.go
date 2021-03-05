package common

//InArgs :
type InArgs struct {
	DirPath   *string
	GitURL    *string
	GitBranch *string
	CommitID  *string
}

//Resource :
type Resource struct {
	GitURL    string `json:"git_url"`
	GitBranch string `json:"git_branch"`
	CommitID  string `json:"commitid"`

	Resource struct {
		Name         string   `json:"name"`
		Selectors    []string `json:"selectors"`
		FilePath     string   `json:"filepath"`
		Kind         string   `json:"kind"`
		ReplicaCount int      `json:"replica,omitempty"`
		Image        struct {
			ID string `json:"id, omitempty"`
		} `json:"image"`
		Network []NetworkAttr `json:"network"`
		Envs    []string
	}
}

//NetworkAttr :
type NetworkAttr struct {
	HostPort      int    `json:"host_port,omitempty"`
	ContainerPort int    `json:"container_url,omitempty"`
	Protocol      string `json:"protocol,omitempty "`
}

//SvcNetworkAttr :
type SvcNetworkAttr struct {
	Port       int    `json:"port,omitempty"`
	TargetPort int    `json:"target_port,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

//Service :
type Service struct {
	GitURL    string `json:"git_url"`
	GitBranch string `json:"git_branch"`
	CommitID  string `json:"commitid"`
	Resource  struct {
		Name      string           `json:"name"`
		Selectors []string         `json:"selectors"`
		FilePath  string           `json:"filepath"`
		Kind      string           `json:"kind"`
		Network   []SvcNetworkAttr `json:"network"`
	} `json:"resource"`
}

//Connections :
type Connections struct {
	Source Resource `json:"source, omitempty"`
	Target Resource `json:"target"`
	Link   Service  `json:"link"`
}

const (
	//ServiceCtx :
	ServiceCtx = "service"

	//DeployCtx :
	DeployCtx = "deployment"
)
