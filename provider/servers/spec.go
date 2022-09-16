package servers

type (
	// Certificate describes TLS certifications.
	Certificate struct {
		Name string `json:"name"`
		Cert string `json:"cert"`
		Key  string `json:"key"`
	}
)
