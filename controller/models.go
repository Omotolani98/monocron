package controller

type ghPushEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
	} `json:"repository"`
	HeadCommit struct {
		ID string `json:"id"`
	} `json:"head_commit"`
	Commits []struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
}

type fetchedFile struct {
	Path          string `json:"path"`
	RefSHA        string `json:"ref_sha"`
	Content       string `json:"content"`
	ContentSHA256 string `json:"content_sha256"`
}

type webhookResponse struct {
	Owner      string        `json:"owner"`
	Repo       string        `json:"repo"`
	HeadSHA    string        `json:"head_sha"`
	FilesCount int           `json:"files_count"`
	Files      []fetchedFile `json:"files"`
}

type Job struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       JobSpec  `yaml:"spec" json:"spec"`
}

type Metadata struct {
	Name string `yaml:"name" json:"name"`
}

type JobSpec struct {
	Schedule []string     `yaml:"schedule" json:"schedule"`
	Timezone string       `yaml:"timezone" json:"timezone"`
	Timeout  string       `yaml:"timeout" json:"timeout"`
	Executor ExecutorSpec `yaml:"executor" json:"executor"`
}

type ExecutorSpec struct {
	Command []string `yaml:"command" json:"command"`
}
